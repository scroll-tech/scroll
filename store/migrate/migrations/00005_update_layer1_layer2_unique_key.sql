-- +goose Up
-- +goose StatementBegin
    drop index layer1_message_nonce_uindex;

    create unique index layer1_message_layer1_hash_uindex
    on layer1_message (layer1_hash);

    drop index layer2_message_nonce_uindex;

    create unique index layer2_message_layer2_hash_uindex
    on layer2_message (layer2_hash);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
    drop index layer1_message_layer1_hash_uindex;

    create unique index layer1_message_nonce_uindex
    on layer1_message (nonce);

    drop index layer2_message_layer2_hash_uindex;

    create unique index layer2_message_nonce_uindex
    on layer2_message (nonce);
-- +goose StatementEnd


