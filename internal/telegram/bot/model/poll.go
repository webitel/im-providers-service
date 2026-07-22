package model

type PollType string

const (
	PollTypeRegular PollType = "regular"
	PollTypeQuiz    PollType = "quiz"
)

// https://core.telegram.org/bots/api#poll
type Poll struct {
	ID                    string          `json:"id"`
	Question              string          `json:"question"`
	QuestionEntities      []MessageEntity `json:"question_entities,omitempty"`
	Options               []PollOption    `json:"options"`
	TotalVoterCount       int64           `json:"total_voter_count"`
	IsClosed              bool            `json:"is_closed"`
	IsAnonymous           bool            `json:"is_anonymous"`
	Type                  PollType        `json:"type"`
	AllowsMultipleAnswers bool            `json:"allows_multiple_answers"`
	AllowsRevoting        bool            `json:"allows_revoting"`
	MembersOnly           bool            `json:"members_only"`
	CountryCodes          []string        `json:"country_codes,omitempty"`
	CorrectOptionIDs      []int64         `json:"correct_option_ids,omitempty"`
	Explanation           *string         `json:"explanation,omitempty"`
	ExplanationEntities   []MessageEntity `json:"explanation_entities,omitempty"`
	ExplanationMedia      *PollMedia      `json:"explanation_media,omitempty"`
	OpenPeriod            *int64          `json:"open_period,omitempty"`
	CloseDate             *int64          `json:"close_date,omitempty"`
	Description           *string         `json:"description,omitempty"`
	DescriptionEntities   []MessageEntity `json:"description_entities,omitempty"`
	Media                 *PollMedia      `json:"media,omitempty"`
}

// https://core.telegram.org/bots/api#polloption
type PollOption struct {
	PersistentID string          `json:"persistent_id"`
	Text         string          `json:"text"`
	TextEntities []MessageEntity `json:"text_entities,omitempty"`
	Media        *PollMedia      `json:"media,omitempty"`
	VoterCount   int64           `json:"voter_count"`
	AddedByUser  *User           `json:"added_by_user,omitempty"`
	AddedByChat  *Chat           `json:"added_by_chat,omitempty"`
	AdditionDate *int64          `json:"addition_date,omitempty"`
}

// https://core.telegram.org/bots/api#pollmedia
// At most one of the optional fields can be present.
type PollMedia struct {
	Animation *Animation  `json:"animation,omitempty"`
	Audio     *Audio      `json:"audio,omitempty"`
	Document  *Document   `json:"document,omitempty"`
	Link      *Link       `json:"link,omitempty"`
	LivePhoto *LivePhoto  `json:"live_photo,omitempty"`
	Location  *Location   `json:"location,omitempty"`
	Photo     []PhotoSize `json:"photo,omitempty"`
	Sticker   *Sticker    `json:"sticker,omitempty"`
	Venue     *Venue      `json:"venue,omitempty"`
	Video     *Video      `json:"video,omitempty"`
}


// https://core.telegram.org/bots/api#polloptionadded
type PollOptionAdded struct {
	PollMessage        *MaybeInaccessibleMessage `json:"poll_message,omitempty"`
	OptionPersistentID string                    `json:"option_persistent_id"`
	OptionText         string                    `json:"option_text"`
	OptionTextEntities []MessageEntity           `json:"option_text_entities,omitempty"`
}

// https://core.telegram.org/bots/api#polloptiondeleted
type PollOptionDeleted struct {
	PollMessage        *MaybeInaccessibleMessage `json:"poll_message,omitempty"`
	OptionPersistentID string                    `json:"option_persistent_id"`
	OptionText         string                    `json:"option_text"`
	OptionTextEntities []MessageEntity           `json:"option_text_entities,omitempty"`
}


type PollAnswer struct {
	
}

// TODO: fill in fields per https://core.telegram.org/bots/api as each is needed.
type (
	Link                     struct{}
	LivePhoto                struct{}
	Venue                    struct{}
	MaybeInaccessibleMessage struct{}
)
