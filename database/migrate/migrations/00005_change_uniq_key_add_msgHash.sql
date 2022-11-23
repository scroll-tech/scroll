-- +goose Up
-- +goose StatementBegin
alter table l1_message add msg_hash varchar not null;

create unique index l1_message_hash_uindex
on l1_message (msg_hash);

create unique index l1_message_nonce_uindex
on l1_message (nonce);

drop index l1_message_layer1_hash_uindex;

alter table l2_message add msg_hash varchar not null;

create unique index l2_message_hash_uindex
on l2_message (msg_hash);

create unique index l2_message_nonce_uindex
on l2_message (nonce);

drop index l2_message_layer2_hash_uindex;
-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
drop index l1_message_hash_uindex;

drop index l1_message_nonce_uindex;

create unique index l1_message_layer1_hash_uindex
on l1_message (layer1_hash);

alter table l1_message drop msg_hash;

drop index l2_message_hash_uindex;

drop index l2_message_nonce_uindex;

create unique index l2_message_layer2_hash_uindex
on l2_message (layer2_hash);

alter table l2_message drop msg_hash;
-- +goose StatementEnd
