-- +goose Up
ALTER TABLE im_provider.gate_message_templates
    RENAME TO sys_msg_templates;

-- +goose Down
ALTER TABLE im_provider.sys_msg_templates
    RENAME TO gate_message_templates;
