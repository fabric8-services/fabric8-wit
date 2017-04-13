-- users
INSERT INTO
   users(created_at, updated_at, id, email, full_name, image_url, bio, url, context_information)
VALUES
   (
      now(), now(), 'f03f023b-0427-4cdb-924b-fb2369018aa7', 'testtwo@example.com', 'test one', 'https://www.gravatar.com/avatar/testtwo', 'my test bio one', 'http://example.com', '{"key": "value"}'
   ),
   (
      now(), now(), 'f03f023b-0427-4cdb-924b-fb2369018aa6', 'testtwo@example.com', 'test two', 'http://https://www.gravatar.com/avatar/testtwo', 'my test bio two', 'http://example.com', '{"key": "value"}'
   )
;
-- identities
INSERT INTO
   identities(created_at, updated_at, id, username, provider_type, user_id, profile_url)
VALUES
   (
      now(), now(), '2a808366-9525-4646-9c80-ed704b2eebde', 'testtwo', 'github', 'f03f023b-0427-4cdb-924b-fb2369018aa7', 'http://example-github.com'
   ),
   (
      now(), now(), '2a808366-9525-4646-9c80-ed704b2eebdb', 'testwo', 'rhhd', 'f03f023b-0427-4cdb-924b-fb2369018aa6', 'http://example-rhd.com'
   )
;