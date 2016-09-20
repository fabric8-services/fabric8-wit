#Deploy ALM-Core using ansible-container

Install ansible-container

```
$ sudo pip install --upgrade setuptools
$ sudo pip install ansible-container
```

Increase the docker-compose read timeout

```
export DOCKER_CLIENT_TIMEOUT=120
export COMPOSE_HTTP_TIMEOUT=120

```

Create the alm-binary from source 

```
$ cd $GOPATH/src/github.com/almighty/almighty-core/deploy/ansible
$ ansible-container build

No DOCKER_HOST environment variable found. Assuming UNIX socket at /var/run/docker.sock
Starting Docker Compose engine to build your images...
The DOCKER_CLIENT_TIMEOUT environment variable is deprecated.  Please use COMPOSE_HTTP_TIMEOUT instead.
0.1: Pulling from ansible/ansible-container-builder
8ad8b3f87b37: Pull complete
751fe39c4d34: Pull complete
ae3b77eefc06: Pull complete
7783aac582ec: Pull complete
b7fc86da4ddd: Pull complete
0c2a6373dba5: Pull complete
7efd1a6c2b0b: Pull complete
ce2333248474: Pull complete
ecb0bbc89afa: Pull complete
e3d0408851f1: Pull complete
6da208a6a5ba: Pull complete
9644f0bbcd04: Pull complete
Digest: sha256:a0e8723656ff176a15db26d0a5212191e521cb9191cc2cb247d826762bc93757
Status: Downloaded newer image for ansible/ansible-container-builder:0.1
Attaching to ansible_ansible-container_1
Cleaning up Ansible Container builder...
No image found for tag alm-app:latest, so building from scratch
No image found for tag alm-database:latest, so building from scratch
The DOCKER_CLIENT_TIMEOUT environment variable is deprecated.  Please use COMPOSE_HTTP_TIMEOUT instead.
9.5: Pulling from library/postgres
8ad8b3f87b37: Already exists
c5f4a4b21ab6: Pull complete
ba05db8b0a52: Pull complete
47b491cd21ab: Pull complete
d70407e3e64d: Pull complete
295c246dd69f: Pull complete
89bc4bb8bcfd: Pull complete
106ff44c5f06: Pull complete
867cd91e76bb: Pull complete
a227948d6d8c: Pull complete
fc2ec20bdaf0: Pull complete
Digest: sha256:1115f095242a490cb79561124a79125e25b0595d5ae47d44fab5b4c1cd10735f
Status: Downloaded newer image for postgres:9.5
7: Pulling from library/centos
8d30e94188e7: Pull complete
Digest: sha256:2ae0d2c881c7123870114fb9cc7afabd1e31f9888dac8286884f6cf59373ed9b
Status: Downloaded newer image for centos:7
Attaching to ansible_ansible-container_1, ansible_database_1, ansible_app_1
ansible-container_1  | 
ansible-container_1  | PLAY [app] *********************************************************************
ansible-container_1  | 
ansible-container_1  | TASK [setup] *******************************************************************
database_1           | The files belonging to this database system will be owned by user "postgres".
database_1           | This user must also own the server process.
database_1           | 
database_1           | The database cluster will be initialized with locale "en_US.utf8".
database_1           | The default database encoding has accordingly been set to "UTF8".
database_1           | The default text search configuration will be set to "english".
database_1           | 
database_1           | Data page checksums are disabled.
database_1           | 
database_1           | fixing permissions on existing directory /var/lib/postgresql/data ... ok
database_1           | creating subdirectories ... ok
database_1           | selecting default max_connections ... 100
database_1           | selecting default shared_buffers ... 128MB
database_1           | selecting dynamic shared memory implementation ... posix
database_1           | creating configuration files ... ok
database_1           | creating template1 database in /var/lib/postgresql/data/base/1 ... ok
database_1           | initializing pg_authid ... ok
ansible-container_1  | ok: [app]
ansible-container_1  | 
ansible-container_1  | TASK [Install the packages] ****************************************************
database_1           | initializing dependencies ... ok
database_1           | creating system views ... ok
database_1           | loading system objects' descriptions ... ok
database_1           | creating collations ... ok
database_1           | creating conversions ... ok
database_1           | creating dictionaries ... ok
database_1           | setting privileges on built-in objects ... ok
database_1           | creating information schema ... ok
database_1           | loading PL/pgSQL server-side language ... ok
database_1           | vacuuming database template1 ... ok
database_1           | copying template1 to template0 ... ok
database_1           | copying template1 to postgres ... ok
database_1           | syncing data to disk ... ok
database_1           | 
database_1           | WARNING: enabling "trust" authentication for local connections
database_1           | You can change this by editing pg_hba.conf or using the option -A, or
database_1           | --auth-local and --auth-host, the next time you run initdb.
database_1           | 
database_1           | Success. You can now start the database server using:
database_1           | 
database_1           |     pg_ctl -D /var/lib/postgresql/data -l logfile start
database_1           | 
database_1           | waiting for server to start....LOG:  database system was shut down at 2016-09-20 07:28:40 UTC
database_1           | LOG:  MultiXact member wraparound protections are now enabled
database_1           | LOG:  database system is ready to accept connections
database_1           | LOG:  autovacuum launcher started
database_1           |  done
database_1           | server started
database_1           | ALTER ROLE
database_1           | 
database_1           | 
database_1           | /docker-entrypoint.sh: ignoring /docker-entrypoint-initdb.d/*
database_1           | 
database_1           | LOG:  received fast shutdown request
database_1           | LOG:  aborting any active transactions
database_1           | LOG:  autovacuum launcher shutting down
database_1           | LOG:  shutting down
database_1           | waiting for server to shut down....LOG:  database system is shut down
database_1           |  done
database_1           | server stopped
database_1           | 
database_1           | PostgreSQL init process complete; ready for start up.
database_1           | 
database_1           | LOG:  database system was shut down at 2016-09-20 07:28:44 UTC
database_1           | LOG:  MultiXact member wraparound protections are now enabled
database_1           | LOG:  database system is ready to accept connections
database_1           | LOG:  autovacuum launcher started
ansible-container_1  | changed: [app] => (item=[u'findutils', u'git', u'golang', u'make', u'mercurial', u'procps-ng', u'tar', u'wget', u'which'])
ansible-container_1  | 
ansible-container_1  | TASK [Get glide for Go package management] *************************************
ansible-container_1  | changed: [app]
ansible-container_1  | 
ansible-container_1  | TASK [Untar the file] **********************************************************
ansible-container_1  | changed: [app]
ansible-container_1  | 
ansible-container_1  | TASK [Make working directory] **************************************************
ansible-container_1  | changed: [app]
ansible-container_1  |  [WARNING]: Consider using file module with state=directory rather than running
ansible-container_1  | mkdir
ansible-container_1  | 
ansible-container_1  | TASK [Set GOPATH] **************************************************************
ansible-container_1  | changed: [app]
ansible-container_1  | 
ansible-container_1  | TASK [Export GOPATH] ***********************************************************
ansible-container_1  | changed: [app]
ansible-container_1  | 
ansible-container_1  | TASK [Git clone the repo] ******************************************************
ansible-container_1  | changed: [app]
ansible-container_1  | 
ansible-container_1  | TASK [Make build] **************************************************************
ansible-container_1  | changed: [app]
ansible-container_1  | 
ansible-container_1  | PLAY RECAP *********************************************************************
ansible-container_1  | app                        : ok=9    changed=8    unreachable=0    failed=0   
ansible-container_1  | 
ansible_ansible-container_1 exited with code 0
Aborting on container exit...
Stopping ansible_app_1 ... done
Stopping ansible_database_1 ... done
Exporting built containers as images...
Committing image...
Exported alm-app with image ID sha256:607ea520fc0c3497398359d1b5a277fe5b5b4a08f98c34fb7a6cbd29324211ed
Cleaning up app build container...
Cleaning up Ansible Container builder...

```
Once the build process is completed all the images required by alm will be created

Verify all the images creted by the build process

```
$ docker images 

REPOSITORY                          TAG                 IMAGE ID            CREATED             SIZE
alm-app                             20160920061043      bed282324bd4        9 seconds ago       828 MB
ansible/ansible-container-builder   0.1                 5613bb4d186e        2 days ago          831.1 MB
centos                              7                   980e0e4c79ec        13 days ago         196.7 MB
postgres                            9.5                 6f86882e145d        2 weeks ago         265.9 MB
```

Deploy the application

```
$ ansible-container run

No DOCKER_HOST environment variable found. Assuming UNIX socket at /var/run/docker.sock
The DOCKER_CLIENT_TIMEOUT environment variable is deprecated.  Please use COMPOSE_HTTP_TIMEOUT instead.
Attaching to ansible_ansible-container_1
Cleaning up Ansible Container builder...
The DOCKER_CLIENT_TIMEOUT environment variable is deprecated.  Please use COMPOSE_HTTP_TIMEOUT instead.
Attaching to ansible_database_1, ansible_app_1
database_1           | LOG:  database system was shut down at 2016-09-20 07:35:30 UTC
database_1           | LOG:  MultiXact member wraparound protections are now enabled
database_1           | LOG:  database system is ready to accept connections
database_1           | LOG:  autovacuum launcher started
app_1                | Running as user name "root" with UID 0.
app_1                | Opening DB connection attempt 1 of 50
app_1                | 2016/09/20 07:41:52 loading work item type system.userstory
app_1                | 2016/09/20 07:41:52 not found, res={{0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC <nil>}  0  map[]}
app_1                | 2016/09/20 07:41:52 loading work item type system.valueproposition
app_1                | 2016/09/20 07:41:52 not found, res={{0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC <nil>}  0  map[]}
app_1                | 2016/09/20 07:41:52 loading work item type system.fundamental
app_1                | 2016/09/20 07:41:52 not found, res={{0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC <nil>}  0  map[]}
app_1                | 2016/09/20 07:41:52 loading work item type system.experience
app_1                | 2016/09/20 07:41:52 not found, res={{0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC <nil>}  0  map[]}
app_1                | 2016/09/20 07:41:52 loading work item type system.feature
app_1                | 2016/09/20 07:41:52 not found, res={{0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC <nil>}  0  map[]}
app_1                | 2016/09/20 07:41:52 loading work item type system.bug
app_1                | 2016/09/20 07:41:52 not found, res={{0001-01-01 00:00:00 +0000 UTC 0001-01-01 00:00:00 +0000 UTC <nil>}  0  map[]}
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Login action=Authorize route=GET /api/login/authorize
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Login action=Generate route=GET /api/login/generate
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Version action=Show route=GET /api/version
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Workitem action=Create route=POST /api/workitems
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Workitem action=Delete route=DELETE /api/workitems/:id
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Workitem action=List route=GET /api/workitems
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Workitem action=Show route=GET /api/workitems/:id
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Workitem action=Update route=PUT /api/workitems/:id
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Workitemtype action=Create route=POST /api/workitemtypes
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Workitemtype action=List route=GET /api/workitemtypes
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Workitemtype action=Show route=GET /api/workitemtypes/:name
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Tracker action=Create route=POST /api/trackers
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Tracker action=Delete route=DELETE /api/trackers/:id
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Tracker action=List route=GET /api/trackers
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Tracker action=Show route=GET /api/trackers/:id
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Tracker action=Update route=PUT /api/trackers/:id
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Trackerquery action=Create route=POST /api/trackerqueries
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Trackerquery action=Show route=GET /api/trackerqueries/:id
app_1                | 2016/09/20 07:41:52 [INFO] mount ctrl=Trackerquery action=Update route=PUT /api/trackerqueries/:id
app_1                | Git Commit SHA:  36a84113df1fa3110462f37504492d6a099703b2
app_1                | UTC Build Time:  2016-09-20_07:34:54AM
app_1                | Dev mode:        false


```

Verify the running containers

```
$ docker ps

CONTAINER ID        IMAGE               COMMAND                  CREATED              STATUS              PORTS                    NAMES
02a493efe1b6        alm-app:latest      "/usr/bin/alm -dbhost"   About a minute ago   Up About a minute   0.0.0.0:8080->8080/tcp   ansible_app_1
1da867631482        postgres:9.5        "/docker-entrypoint.s"   14 minutes ago       Up About a minute   0.0.0.0:5432->5432/tcp   ansible_database_1

```

Verify the application
```
$ curl localhost:8080/api/workitemtypes

[{"fields":{"system.assignee":{"required":false,"type":{"kind":"user"}},"system.creator":{"required":true,"type":{"kind":"user"}},"system.description":{"required":false,"type":{"kind":"string"}},"system.state":{"required":true,"type":{"baseType":"string","kind":"enum","values":["new","in progress","resolved","closed"]}},"system.title":{"required":true,"type":{"kind":"string"}}},"name":"system.userstory","version":0},{"fields":{"system.assignee":{"required":false,"type":{"kind":"user"}},"system.creator":{"required":true,"type":{"kind":"user"}},"system.description":{"required":false,"type":{"kind":"string"}},"system.state":{"required":true,"type":{"baseType":"string","kind":"enum","values":["new","in progress","resolved","closed"]}},"system.title":{"required":true,"type":{"kind":"string"}}},"name":"system.valueproposition","version":0},{"fields":{"system.assignee":{"required":false,"type":{"kind":"user"}},"system.creator":{"required":true,"type":{"kind":"user"}},"system.description":{"required":false,"type":{"kind":"string"}},"system.state":{"required":true,"type":{"baseType":"string","kind":"enum","values":["new","in progress","resolved","closed"]}},"system.title":{"required":true,"type":{"kind":"string"}}},"name":"system.fundamental","version":0},{"fields":{"system.assignee":{"required":false,"type":{"kind":"user"}},"system.creator":{"required":true,"type":{"kind":"user"}},"system.description":{"required":false,"type":{"kind":"string"}},"system.state":{"required":true,"type":{"baseType":"string","kind":"enum","values":["new","in progress","resolved","closed"]}},"system.title":{"required":true,"type":{"kind":"string"}}},"name":"system.experience","version":0},{"fields":{"system.assignee":{"required":false,"type":{"kind":"user"}},"system.creator":{"required":true,"type":{"kind":"user"}},"system.description":{"required":false,"type":{"kind":"string"}},"system.state":{"required":true,"type":{"baseType":"string","kind":"enum","values":["new","in progress","resolved","closed"]}},"system.title":{"required":true,"type":{"kind":"string"}}},"name":"system.feature","version":0},{"fields":{"system.assignee":{"required":false,"type":{"kind":"user"}},"system.creator":{"required":true,"type":{"kind":"user"}},"system.description":{"required":false,"type":{"kind":"string"}},"system.state":{"required":true,"type":{"baseType":"string","kind":"enum","values":["new","in progress","resolved","closed"]}},"system.title":{"required":true,"type":{"kind":"string"}}},"name":"system.bug","version":0}]
```

