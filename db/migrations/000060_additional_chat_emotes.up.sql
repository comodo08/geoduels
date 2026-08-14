alter table chat_messages
  drop constraint if exists chat_messages_text_body_check;

alter table chat_messages
  add constraint chat_messages_text_body_check
  check (
    (kind = 'text' and body is not null and length(body) > 0 and emote is null)
    or
    (kind = 'emote' and emote in ('greetings', 'skull', 'sob', 'thinking', 'sunglasses', 'laugh', 'pray') and body is null)
  );
