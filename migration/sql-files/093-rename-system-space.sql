-- rename the `system.space` to `system-space` to comply with the system naming pattern which
-- does not allow the `.` character
update spaces set name = 'system-space' where id = '2e0698d8-753e-4cef-bb7c-f027634824a2';