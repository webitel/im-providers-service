package model

// https://core.telegram.org/bots/api#user
type User struct {
	ID                         int64   `json:"id"`
	IsBot                      bool    `json:"is_bot"`
	FirstName                  string  `json:"first_name"`
	LastName                   *string `json:"last_name,omitempty"`
	Username                   *string `json:"username,omitempty"`
	LanguageCode               *string `json:"language_code,omitempty"`
	IsPremium                  *bool   `json:"is_premium,omitempty"`
	AddedToAttachmentMenu      *bool   `json:"added_to_attachment_menu,omitempty"`
	CanJoinGroups              *bool   `json:"can_join_groups,omitempty"`
	CanReadAllGroupMessages    *bool   `json:"can_read_all_group_messages,omitempty"`
	SupportsGuestQueries       *bool   `json:"supports_guest_queries,omitempty"`
	SupportsInlineQueries      *bool   `json:"supports_inline_queries,omitempty"`
	CanConnectToBusiness       *bool   `json:"can_connect_to_business,omitempty"`
	HasMainWebApp              *bool   `json:"has_main_web_app,omitempty"`
	HasTopicsEnabled           *bool   `json:"has_topics_enabled,omitempty"`
	AllowsUsersToCreateTopics  *bool   `json:"allows_users_to_create_topics,omitempty"`
	CanManageBots              *bool   `json:"can_manage_bots,omitempty"`
	SupportsJoinRequestQueries *bool   `json:"supports_join_request_queries,omitempty"`
}
