package domain

// Policy represents a policy resource that can be linked to grants and inboxes.
type Policy struct {
	ID             string               `json:"id,omitempty"`
	Name           string               `json:"name,omitempty"`
	ApplicationID  string               `json:"application_id,omitempty"`
	OrganizationID string               `json:"organization_id,omitempty"`
	Rules          []string             `json:"rules,omitempty"`
	Limits         *PolicyLimits        `json:"limits,omitempty"`
	Options        *PolicyOptions       `json:"options,omitempty"`
	SpamDetection  *PolicySpamDetection `json:"spam_detection,omitempty"`
	CreatedAt      UnixTime             `json:"created_at,omitempty"`
	UpdatedAt      UnixTime             `json:"updated_at,omitempty"`
}

// PolicyLimits contains limit settings for a policy.
type PolicyLimits struct {
	LimitAttachmentSizeInBytes      *int64    `json:"limit_attachment_size_limit,omitempty"`
	LimitAttachmentCount            *int      `json:"limit_attachment_count_limit,omitempty"`
	LimitAttachmentAllowedTypes     *[]string `json:"limit_attachment_allowed_types,omitempty"`
	LimitSizeTotalMimeInBytes       *int64    `json:"limit_size_total_mime,omitempty"`
	LimitStorageTotalInBytes        *int64    `json:"limit_storage_total,omitempty"`
	LimitCountDailyMessagePerGrant  *int64    `json:"limit_count_daily_message_per_grant,omitempty"`
	LimitInboxRetentionPeriodInDays *int      `json:"limit_inbox_retention_period,omitempty"`
	LimitSpamRetentionPeriodInDays  *int      `json:"limit_spam_retention_period,omitempty"`
}

// PolicyOptions contains option settings for a policy.
type PolicyOptions struct {
	AdditionalFolders *[]string `json:"additional_folders,omitempty"`
	UseCidrAliasing   *bool     `json:"use_cidr_aliasing,omitempty"`
}

// PolicySpamDetection contains spam detection settings for a policy.
type PolicySpamDetection struct {
	UseListDNSBL              *bool    `json:"use_list_dnsbl,omitempty"`
	UseHeaderAnomalyDetection *bool    `json:"use_header_anomaly_detection,omitempty"`
	SpamSensitivity           *float64 `json:"spam_sensitivity,omitempty"`
}
