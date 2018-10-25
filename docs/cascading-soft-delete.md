# Cascading soft-deletes

## Abstract

In this document I describe our current approach to deal with deletion of objects in our Postgres database and I present the flaws of the current implementation. For example so far we don't have the ability to have cascading soft-deletes. Then I show a method that combines the strengths of Postgres' **cascading soft-delete** and an archiving approach that is easy to implement, maintain and that brings a performance boost in all search queries.

## About soft-deletes in GORM

In the [fabric8-services/fabric8-wit](https://github.com/fabric8-services/fabric8-wit) project which is written in Go we are using the an object oriented mapper for our database called [GORM](http://gorm.io/).

GORM offers a way to [soft-delete](http://gorm.io/docs/delete.html#Soft-Delete) database entries:

> If model has `DeletedAt` field, it will get soft delete ability automatically! then it won’t be deleted from database permanently when call `Delete`, but only set field `DeletedAt`‘s value to current time.

Suppose you have a model definition, in other words a Go struct that looks like this:

```go
// User is the Go model for a user entry in the database
type User struct {
	ID        int
	Name      string
  DeletedAt *time.Time
}
```

And let's say you've loaded an existing user entry by its `ID` from the DB into an object `u`.

```go
id := 123
u := User{}
db.Where("id=?", id).First(&u)
```

If you then go ahead and delete the object using GORM:

```go
db.Delete(&u)
```

the DB entry will not be deleted using `DELETE` in SQL but the row will be updated and the `deleted_at` is set to the current time:

```sql
UPDATE users SET deleted_at="2018-10-12 11:24" WHERE id = 123;
```

## Problems with soft-deletes in GORM - Inversion of dependency and no cascade

The above mentioned soft-delete is nice for archiving individual records but it can lead to very odd results for all records that depend on it. That is because soft-deletes by GORM don't cascade as a potential `DELETE` in SQL would do if a foreign key was modelled with `ON DELETE CASCADE`.

When you model a database you typcially design a table and then maybe another one that has a foreign key to the first one:

```sql
CREATE TABLE countries (
    name text PRIMARY KEY,
    deleted_at timestamp
);

CREATE TABLE cities (
    name text,
    country text REFERENCES countries(name) ON DELETE CASCADE,
    deleted_at timestamp
);
```

Here we've modeled a list of countries and cities that reference a particular country. When you `DELETE` a country record, all cities will be deleted as well. But since the table has a `deleted_at` column that is carried on in the Go struct for a country or city, the GORM mapper will only soft-delete the country and leave the belonging cities untouched.

### Shifting responsibility from DB to user/developer

GORM thereby puts it in the hands of the developer to (soft-)delete all dependend cities. In other words, what previously was modeled as a relationship from **cities to countries** is now reversed as a relationship from **countries to cities**. That is because the user/developer is now responsible to (soft-)delete all cities belonging to a country when the country is deleted.

## Proposal

Wouldn't it be great if we can have soft-deletes and all the benefits of a `ON DELETE CASCADE`?

It turns out that we can have it without much effort. Let's focus on a single table for now, namely the `countries` table. 

### An archive table

Suppose for a second, that we can have another table called `countries_archive` that has the excact **same structure** as the `countries` table. Also suppose that all future **schema migrations** that are done to `countries` are applied to the `countries_archive` table. The only exception is that **unique constraints** and **foreign keys** will not be applied to `countries_archive`.

I guess, this already sounds too good to be true, right? Well, we can create such a table using what's called [Inheritenance](https://www.postgresql.org/docs/9.6/static/ddl-inherit.html) in Postgres:

```sql
CREATE TABLE countries_archive () INHERITS (countries);
```

The resulting `countries_archive` table will is meant to store all records where `deleted_at IS NOT NULL`.

Note, that in our Go code we would never directly use any `_archive` table. Instead we would query for the original table from which `*_archive` table inherits and Postgres then magically looks into the `*_archive` table automatically. A bit further below I explain why that is; it has to do with partitioning.

### Moving entries to the archive table on (soft)-DELETE

Since the two tables, `countries` and `countries_archive` look exactly alike schemawise we can `INSERT` into the archive very easily using a trigger function when

1. a `DELETE` happens on the `countries` table
2. or when a soft-delete is happening by setting `deleted_at` to a not `NULL` value.

The trigger function looks like this:

```sql
CREATE OR REPLACE FUNCTION archive_record()
RETURNS TRIGGER AS $$
BEGIN
    -- When a soft-delete happens...
    IF (TG_OP = 'UPDATE' AND NEW.deleted_at IS NOT NULL) THEN
        EXECUTE format('DELETE FROM %I.%I WHERE id = $1', TG_TABLE_SCHEMA, TG_TABLE_NAME) USING OLD.id;
        RETURN OLD;
    END IF;
    -- When a hard-DELETE or a cascaded delete happens
    IF (TG_OP = 'DELETE') THEN
        -- Set the time when the deletion happens
        IF (OLD.deleted_at IS NULL) THEN
            OLD.deleted_at := timenow();
        END IF;
        EXECUTE format('INSERT INTO %I.%I SELECT $1.*'
                    , TG_TABLE_SCHEMA, TG_TABLE_NAME || '_archive')
        USING OLD;
    END IF;
    RETURN NULL;
END;
  $$ LANGUAGE plpgsql;
```

To wire-up the function with a trigger we can write:

```sql
CREATE TRIGGER soft_delete_countries
    AFTER
        -- this is what is triggered by GORM
        UPDATE OF deleted_at 
        -- this is what is triggered by a cascaded DELETE or a direct hard-DELETE
        OR DELETE
    ON countries
    FOR EACH ROW
    EXECUTE PROCEDURE archive_record();
```

## Conclusions

Originally the inheritance functionality in postgres was developed to [partition data](https://www.postgresql.org/docs/9.1/static/ddl-partitioning.html). When you search your partitioned data using a specific column or condition, Postgres can find out which partition to search through and can thereby [improve the performance of your query](https://stackoverflow.com/a/3075248/835098).

We can benefit from this performance improvement by only searching entities in existence, unless told otherwise. Entries in existence are those where `deleted_at IS NULL` holds true. (Notice, that GORM will automatically add an `AND deleted_at IS NULL` to every query if there's a `DeletedAt` in GORM's model struct.)

Let's see if Postgres already knows how to take advantage of our separation by running an `EXPLAIN`:

```sql
EXPLAIN SELECT * FROM countries WHERE deleted_at IS NULL;
+-------------------------------------------------------------------------+
| QUERY PLAN                                                              |
|-------------------------------------------------------------------------|
| Append  (cost=0.00..21.30 rows=7 width=44)                              |
|   ->  Seq Scan on countries  (cost=0.00..0.00 rows=1 width=44)          |
|         Filter: (deleted_at IS NULL)                                    |
|   ->  Seq Scan on countries_archive  (cost=0.00..21.30 rows=6 width=44) |
|         Filter: (deleted_at IS NULL)                                    |
+-------------------------------------------------------------------------+
```

As we can see, Postgres still searches both tables, `countries` and `countries_archive`. Let's see what happens when we add a check constraint to the `countries_archive` table upon table creation:

```sql
CREATE TABLE countries_archive (
    CHECK (deleted_at IS NOT NULL)
) INHERITS (countries);
```

Now, Postgres knows that it can skip `countries_archive` when `deleted_at` is expected to be `NULL`:

```sql
EXPLAIN SELECT * FROM countries WHERE deleted_at IS NULL;
+----------------------------------------------------------------+
| QUERY PLAN                                                     |
|----------------------------------------------------------------|
| Append  (cost=0.00..0.00 rows=1 width=44)                      |
|   ->  Seq Scan on countries  (cost=0.00..0.00 rows=1 width=44) |
|         Filter: (deleted_at IS NULL)                           |
+----------------------------------------------------------------+
```

Notice the absence of the sequential scan of the `countries_archive` table in the aforementioned `EXPLAIN`.

## Benefits

1. We have regular **cascaded deletes** back and can let the DB figure out in which order to delete things.
2. At the same time, we're **archiving our data** as well. Every soft-delete 
3. **No Go code changes** are required. We only have to setup a table and a trigger for each table that shall be archived.
4. Whenever we figure that we don't want this behaviour with triggers and cascaded soft-delete anymore **we can easily go back**.
5. All future **schema migrations** that are being made to the original table will be applied to the `_archive` version of that table as well. Except for constraints, which is good.

## Final thoughts

The approach presented here does not solve the problem of restoring individual rows. On the other hand, this approach doesn't make it harder or more complicated. It just remains unsolved.

In our application certain fields of a work item don't have a foreign key specified. A good example are the area IDs. That means when an area is `DELETE`d, an associated work item is not automatically `DELETE`d. There are two scenarios when an area is removed itself:

1. A delete is directly requested from a user.
2. A user requests to delete a space and then the area is removed due to its foreign key constraint on the space.

Notice that, in the first scenario the user's requests goes through the area controller code and then through the area repository code. We have a chance in any of those layers to modify all work items that would reference a non-existing area otherwise. In the second scenario everything related to the area happens and stays on the DB layer so we have no chance of moifying the work items. The good news is that we don't have to. Every work item references a space and will therefore be deleted anyways when the space goes away.

What applies to areas also applies to iterations, labels and board columns as well.

# How to apply to our database?

Steps

1. Create "*_archived" tables for all tables that inherit the original tables.
2. Install a soft-delete trigger usinge the above `archive_record()` function.
3. Move all entries where `deleted_at IS NOT NULL` to their respective `_archive` table by doing a hard `DELETE` which will trigger the `archive_record()` function.

# Example

Here is a [fully working example](https://rextester.com/BMCB94324) in which we demonstrated a cascaded soft-delete over two tables, `countries` and `capitals`. We show how records are being archived independently of the method that was chosen for the delete.

```sql
CREATE TABLE countries (
    id int primary key,
    name text unique,
    deleted_at timestamp
);
CREATE TABLE countries_archive (
    CHECK ( deleted_at IS NOT NULL )
) INHERITS(countries);

CREATE TABLE capitals (
    id int primary key,
    name text,
    country_id int references countries(id) on delete cascade,
    deleted_at timestamp
);
CREATE TABLE capitals_archive (
    CHECK ( deleted_at IS NOT NULL )
) INHERITS(capitals);

CREATE OR REPLACE FUNCTION archive_record()
RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'UPDATE' AND NEW.deleted_at IS NOT NULL) THEN
        EXECUTE format('DELETE FROM %I.%I WHERE id = $1', TG_TABLE_SCHEMA, TG_TABLE_NAME) USING OLD.id;
        RETURN OLD;
    END IF;
    IF (TG_OP = 'DELETE') THEN
        IF (OLD.deleted_at IS NULL) THEN
            OLD.deleted_at := timenow();
        END IF;
        EXECUTE format('INSERT INTO %I.%I SELECT $1.*'
                    , TG_TABLE_SCHEMA, TG_TABLE_NAME || '_archive')
        USING OLD;
    END IF;
    RETURN NULL;
END;
  $$ LANGUAGE plpgsql;

CREATE TRIGGER soft_delete_countries
    AFTER
        UPDATE OF deleted_at 
        OR DELETE
    ON countries
    FOR EACH ROW
    EXECUTE PROCEDURE archive_record();
    
CREATE TRIGGER soft_delete_capitals
    AFTER
        UPDATE OF deleted_at 
        OR DELETE
    ON capitals
    FOR EACH ROW
    EXECUTE PROCEDURE archive_record();

INSERT INTO countries (id, name) VALUES (1, 'France');
INSERT INTO countries (id, name) VALUES (2, 'India');
INSERT INTO capitals VALUES (1, 'Paris', 1);
INSERT INTO capitals VALUES (2, 'Bengaluru', 2);

SELECT 'BEFORE countries' as "info", * FROM ONLY countries;
SELECT 'BEFORE countries_archive' as "info", * FROM countries_archive;
SELECT 'BEFORE capitals' as "info", * FROM ONLY capitals;
SELECT 'BEFORE capitals_archive' as "info", * FROM capitals_archive;

-- Delete one country via hard-DELETE and one via soft-delete
DELETE FROM countries WHERE id = 1;
UPDATE countries SET deleted_at = '2018-12-01' WHERE id = 2;

SELECT 'AFTER countries' as "info", * FROM ONLY countries;
SELECT 'AFTER countries_archive' as "info", * FROM countries_archive;
SELECT 'AFTER capitals' as "info", * FROM ONLY capitals;
SELECT 'AFTER capitals_archive' as "info", * FROM capitals_archive;
```