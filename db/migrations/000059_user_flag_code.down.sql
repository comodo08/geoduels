-- Reverts 000059_user_flag_code: removes the profile country flag column.
alter table users
  drop column if exists flag_code;
