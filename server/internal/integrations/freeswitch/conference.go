package freeswitch

import (
	"context"
	"strings"
)

func (c *Client) Conference(ctx context.Context, req ConferenceRequest) (ConferenceResult, error) {
	if err := req.Validate(); err != nil {
		return ConferenceResult{}, err
	}

	args := append([]string{req.Name, req.Command}, req.Arguments...)
	reply, err := c.Command(ctx, "conference "+commandWords(args...))
	if err != nil {
		return ConferenceResult{}, err
	}
	if err := commandReplyError(reply); err != nil {
		return ConferenceResult{}, err
	}
	return ConferenceResult(reply), nil
}

func (c *Client) ListConferenceMembers(ctx context.Context, conference string) (ConferenceMembers, error) {
	conference, err := requiredArgument("conference name", conference)
	if err != nil {
		return ConferenceMembers{}, err
	}

	reply, err := c.Conference(ctx, ConferenceRequest{Name: conference, Command: "list"})
	if err != nil {
		return ConferenceMembers{}, err
	}
	return ConferenceMembers{
		Conference: conference,
		Members:    parseConferenceMembers(reply.Body),
	}, nil
}

func (c *Client) MuteMember(ctx context.Context, conference, memberID string) error {
	return c.conferenceMemberCommand(ctx, conference, "mute", memberID)
}

func (c *Client) UnmuteMember(ctx context.Context, conference, memberID string) error {
	return c.conferenceMemberCommand(ctx, conference, "unmute", memberID)
}

func (c *Client) KickMember(ctx context.Context, conference, memberID string) error {
	return c.conferenceMemberCommand(ctx, conference, "kick", memberID)
}

func (c *Client) DeafMember(ctx context.Context, conference, memberID string) error {
	return c.conferenceMemberCommand(ctx, conference, "deaf", memberID)
}

func (c *Client) UndeafMember(ctx context.Context, conference, memberID string) error {
	return c.conferenceMemberCommand(ctx, conference, "undeaf", memberID)
}

func (c *Client) LockConference(ctx context.Context, conference string) error {
	return c.conferenceCommand(ctx, conference, "lock")
}

func (c *Client) UnlockConference(ctx context.Context, conference string) error {
	return c.conferenceCommand(ctx, conference, "unlock")
}

func (c *Client) conferenceMemberCommand(ctx context.Context, conference, command, memberID string) error {
	memberID, err := requiredArgument("conference member ID", memberID)
	if err != nil {
		return err
	}
	return c.conferenceCommand(ctx, conference, command, memberID)
}

func (c *Client) conferenceCommand(ctx context.Context, conference, command string, args ...string) error {
	conference, err := requiredArgument("conference name", conference)
	if err != nil {
		return err
	}
	command, err = requiredArgument("conference command", command)
	if err != nil {
		return err
	}
	_, err = c.Conference(ctx, ConferenceRequest{
		Name:      conference,
		Command:   command,
		Arguments: args,
	})
	return err
}

func parseConferenceMembers(body string) []ConferenceMember {
	var members []ConferenceMember
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ";", 4)
		if len(parts) < 2 {
			continue
		}
		members = append(members, ConferenceMember{
			ID:       parts[0],
			CallerID: parts[1],
		})
	}
	return members
}
