package mastodon

import (
	"time"
)

const rubyTimeFormat = "2006-01-02 15:04:05.000000000 Z07:00"

type DomainBlock struct {
	ID             StringableU64 `json:"id"`
	Domain         string        `json:"domain"`
	PrivateComment NullString    `json:"private_comment"`
	PublicComment  NullString    `json:"public_comment"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Severity       Severity      `json:"severity"`
	RejectMedia    bool          `json:"reject_media"`
	RejectReports  bool          `json:"reject_reports"`
	Obfuscate      bool          `json:"obfuscate"`
}

func (block DomainBlock) AsYAML() string {
	createdAtStr := block.CreatedAt.Format(rubyTimeFormat)
	updatedAtStr := block.UpdatedAt.Format(rubyTimeFormat)

	return yamlToString(
		yamlMakeDoc(
			yamlMakeMap(
				yamlMakeString("id"),
				yamlMakeInt(uint64(block.ID)),
				yamlMakeString("domain"),
				yamlMakeString(block.Domain),
				yamlMakeString("created_at"),
				yamlMakeTaggedMap(
					"!ruby/object:ActiveSupport::TimeWithZone",
					yamlMakeString("utc"),
					yamlMakeString(createdAtStr),
					yamlMakeString("zone"),
					yamlMakeTaggedMap(
						"!ruby/object:ActiveSupport::TimeZone",
						yamlMakeString("name"),
						yamlMakeString("Etc/UTC"),
					),
					yamlMakeString("time"),
					yamlMakeString(createdAtStr),
				),
				yamlMakeString("updated_at"),
				yamlMakeTaggedMap(
					"!ruby/object:ActiveSupport::TimeWithZone",
					yamlMakeString("utc"),
					yamlMakeString(updatedAtStr),
					yamlMakeString("zone"),
					yamlMakeTaggedMap(
						"!ruby/object:ActiveSupport::TimeZone",
						yamlMakeString("name"),
						yamlMakeString("Etc/UTC"),
					),
					yamlMakeString("time"),
					yamlMakeString(updatedAtStr),
				),
				yamlMakeString("severity"),
				yamlMakeString(block.Severity.String()),
				yamlMakeString("reject_media"),
				yamlMakeBool(block.RejectMedia),
				yamlMakeString("reject_reports"),
				yamlMakeBool(block.RejectReports),
				yamlMakeString("private_comment"),
				yamlMakeNullString(block.PrivateComment),
				yamlMakeString("public_comment"),
				yamlMakeNullString(block.PublicComment),
				yamlMakeString("obfuscate"),
				yamlMakeBool(block.Obfuscate),
			),
		),
	)
}
