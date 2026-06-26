package rpcserver

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/nylas/cli/internal/domain"
	"github.com/nylas/cli/internal/ports"
)

const (
	contactPollLimit    = 100
	maxContactPollPages = 500
)

type ContactPoller struct {
	client  ports.ContactClient
	grantID string
	seen    map[string]string
	seeded  bool
	notify  NotifyFunc
}

type contactUpdatedPayload struct {
	ID        string                `json:"id"`
	GivenName string                `json:"given_name"`
	Surname   string                `json:"surname"`
	Emails    []domain.ContactEmail `json:"emails"`
	UpdatedAt int64                 `json:"updated_at"`
}

func NewContactPoller(client ports.ContactClient, grantID string, notify NotifyFunc) *ContactPoller {
	return &ContactPoller{
		client:  client,
		grantID: grantID,
		seen:    make(map[string]string),
		notify:  notify,
	}
}

func (p *ContactPoller) PollOnce(ctx context.Context) error {
	var contacts []domain.Contact
	pageToken := ""
	for page := range maxContactPollPages {
		resp, err := p.client.GetContactsWithCursor(ctx, p.grantID, &domain.ContactQueryParams{
			Limit:     contactPollLimit,
			PageToken: pageToken,
		})
		if err != nil {
			return err
		}
		if resp == nil {
			return errors.New("contact poll response is nil")
		}
		contacts = append(contacts, resp.Data...)
		if resp.Pagination.NextCursor == "" || !resp.Pagination.HasMore {
			break
		}
		if page == maxContactPollPages-1 {
			return fmt.Errorf("contact poll truncated at %d pages; not committing snapshot", maxContactPollPages)
		}
		pageToken = resp.Pagination.NextCursor
	}
	// ponytail: cap full refetches at 500 pages (~50k contacts); truncation errors still backstop pathological pagination bugs.

	nextSeen := make(map[string]string, len(contacts))
	var changed []domain.Contact
	for _, contact := range contacts {
		fingerprint := contactFingerprint(contact)
		nextSeen[contact.ID] = fingerprint
		if !p.seeded {
			continue
		}
		if lastSeen, ok := p.seen[contact.ID]; !ok || lastSeen != fingerprint {
			changed = append(changed, contact)
		}
	}

	for _, contact := range changed {
		if err := p.notify("contact.updated", contactUpdatedPayload{
			ID:        contact.ID,
			GivenName: contact.GivenName,
			Surname:   contact.Surname,
			Emails:    contact.Emails,
			UpdatedAt: contact.UpdatedAt,
		}); err != nil {
			return err
		}
	}

	if p.seeded {
		for id := range p.seen {
			if _, ok := nextSeen[id]; ok {
				continue
			}
			if err := p.notify("contact.deleted", map[string]string{"id": id}); err != nil {
				return err
			}
		}
	}

	p.seen = nextSeen
	p.seeded = true
	return nil
}

func contactFingerprint(c domain.Contact) string {
	records := []string{}
	appendRecord := func(parts ...string) {
		encoded := make([]string, 0, len(parts)*2)
		for _, part := range parts {
			encoded = append(encoded, strconv.Itoa(len(part)), part)
		}
		records = append(records, strings.Join(encoded, ":"))
	}
	appendSorted := func(label string, values []string) {
		sort.Strings(values)
		for _, value := range values {
			appendRecord(label, value)
		}
	}

	appendRecord("given_name", c.GivenName)
	appendRecord("middle_name", c.MiddleName)
	appendRecord("surname", c.Surname)
	appendRecord("suffix", c.Suffix)
	appendRecord("nickname", c.Nickname)
	appendRecord("birthday", c.Birthday)
	appendRecord("company_name", c.CompanyName)
	appendRecord("job_title", c.JobTitle)
	appendRecord("manager_name", c.ManagerName)
	appendRecord("notes", c.Notes)
	appendRecord("picture_url", c.PictureURL)
	appendRecord("picture", c.Picture)
	appendRecord("source", c.Source)

	emails := make([]string, 0, len(c.Emails))
	for _, email := range c.Emails {
		emails = append(emails, strings.Join([]string{email.Email, email.Type}, "\x00"))
	}
	appendSorted("emails", emails)

	phones := make([]string, 0, len(c.PhoneNumbers))
	for _, phone := range c.PhoneNumbers {
		phones = append(phones, strings.Join([]string{phone.Number, phone.Type}, "\x00"))
	}
	appendSorted("phone_numbers", phones)

	webPages := make([]string, 0, len(c.WebPages))
	for _, webPage := range c.WebPages {
		webPages = append(webPages, strings.Join([]string{webPage.URL, webPage.Type}, "\x00"))
	}
	appendSorted("web_pages", webPages)

	imAddresses := make([]string, 0, len(c.IMAddresses))
	for _, im := range c.IMAddresses {
		imAddresses = append(imAddresses, strings.Join([]string{im.IMAddress, im.Type}, "\x00"))
	}
	appendSorted("im_addresses", imAddresses)

	addresses := make([]string, 0, len(c.PhysicalAddresses))
	for _, address := range c.PhysicalAddresses {
		addresses = append(addresses, strings.Join([]string{
			address.Type,
			address.StreetAddress,
			address.City,
			address.State,
			address.PostalCode,
			address.Country,
		}, "\x00"))
	}
	appendSorted("physical_addresses", addresses)

	groups := make([]string, 0, len(c.Groups))
	for _, group := range c.Groups {
		groups = append(groups, group.ID)
	}
	appendSorted("groups", groups)

	sum := sha256.Sum256([]byte(strings.Join(records, "\x00")))
	return fmt.Sprintf("%x", sum)
}
