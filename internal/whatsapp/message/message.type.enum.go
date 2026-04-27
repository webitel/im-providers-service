package message

type NotificationMessageTypeEnum string

const (
	NotificationMessageTypeText     NotificationMessageTypeEnum = "text"
	NotificationMessageTypeImage    NotificationMessageTypeEnum = "image"
	NotificationMessageTypeDocument NotificationMessageTypeEnum = "document"
)
