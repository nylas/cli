package domain

// CleanMessagesMaxIDs is the maximum number of message IDs the clean endpoint
// accepts in a single request.
const CleanMessagesMaxIDs = 20

// CleanMessagesRequest configures a clean-conversation request
// (PUT /v3/grants/{id}/messages/clean).
//
// The boolean options are pointers so that an unset option is omitted from the
// request body and the API default applies. Defaults (when omitted) are:
// IgnoreLinks=true, IgnoreImages=true, IgnoreTables=true, ImagesAsMarkdown=false,
// RemoveConclusionPhrases=true.
type CleanMessagesRequest struct {
	MessageIDs              []string `json:"message_id"`
	IgnoreLinks             *bool    `json:"ignore_links,omitempty"`
	IgnoreImages            *bool    `json:"ignore_images,omitempty"`
	IgnoreTables            *bool    `json:"ignore_tables,omitempty"`
	ImagesAsMarkdown        *bool    `json:"images_as_markdown,omitempty"`
	RemoveConclusionPhrases *bool    `json:"remove_conclusion_phrases,omitempty"`
}

// CleanedMessage is a single message returned by the clean endpoint. The cleaned
// (HTML) message body is in Conversation; Body holds the original message body.
type CleanedMessage struct {
	ID           string `json:"id"`
	GrantID      string `json:"grant_id,omitempty"`
	Object       string `json:"object,omitempty"`
	Subject      string `json:"subject,omitempty"`
	Conversation string `json:"conversation"`
	Body         string `json:"body,omitempty"`
}
