package freeswitch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	playbackPathVariable     = "leamout_playback_path"
	displaceHangupOnErrorVar = "DISPLACE_HANGUP_ON_ERROR"
)

func (c *Client) Originate(ctx context.Context, req OriginateRequest) (Call, error) {
	if err := req.Validate(); err != nil {
		return Call{}, err
	}

	channelID := uuid.NewString()
	variables := make(map[string]string, len(req.Variables)+1)
	for key, value := range req.Variables {
		variables[key] = value
	}
	variables["origination_uuid"] = channelID

	endpoint := req.Endpoint
	if req.CallerID != "" {
		callerIDVar := "origination_caller_id_number=" + commandWord(req.CallerID)
		if prefix := formatVariables(variables); prefix != "" {
			endpoint = strings.TrimSuffix(prefix, "}") + "," + callerIDVar + "}" + endpoint
		} else {
			endpoint = "{" + callerIDVar + "}" + endpoint
		}
	} else if prefix := formatVariables(variables); prefix != "" {
		endpoint = prefix + endpoint
	}

	// Queue origination through bgapi so the API request does not block while
	// the remote destination rings. origination_uuid gives Leamout the channel
	// identity immediately while calls.id remains a separate logical identity.
	command := "originate " + commandWords(endpoint, "&park()")
	if _, err := c.BGAPI(ctx, command); err != nil {
		return Call{}, err
	}

	return Call{
		UUID:        channelID,
		Destination: req.Destination,
	}, nil
}

// Answer answers an incoming call.
func (c *Client) Answer(ctx context.Context, callID string) error {
	callID, err := requiredArgument("call ID", callID)
	if err != nil {
		return err
	}
	return c.commandOK(ctx, "uuid_answer "+commandWord(callID))
}

func (c *Client) Hangup(ctx context.Context, callID string) error {
	callID, err := requiredArgument("call ID", callID)
	if err != nil {
		return err
	}
	return c.commandOK(ctx, "uuid_kill "+commandWord(callID))
}

func (c *Client) Hold(ctx context.Context, callID string) error {
	callID, err := requiredArgument("call ID", callID)
	if err != nil {
		return err
	}
	return c.commandOK(ctx, "uuid_hold "+commandWord(callID))
}

func (c *Client) Unhold(ctx context.Context, callID string) error {
	callID, err := requiredArgument("call ID", callID)
	if err != nil {
		return err
	}
	// FreeSWITCH resumes a held channel through uuid_hold's "off" form; it
	// does not expose a uuid_unhold API command.
	return c.commandOK(ctx, "uuid_hold "+commandWords("off", callID))
}

// PlayAudio attaches playback as a media bug so stopping it does not interrupt
// the park application that owns the ESL-controlled channel.
func (c *Client) PlayAudio(ctx context.Context, callID, filePath string) error {
	callID, err := requiredArgument("call ID", callID)
	if err != nil {
		return err
	}
	filePath, err = requiredArgument("audio file path", filePath)
	if err != nil {
		return err
	}

	// A media-source failure must never tear down an otherwise healthy call.
	// FreeSWITCH's displace_session can hang up the channel on playback errors;
	// disable that behavior before attaching the media bug so the API can report
	// media failure independently from call lifecycle.
	if err := c.SetVariable(ctx, callID, displaceHangupOnErrorVar, "false"); err != nil {
		return err
	}
	if err := c.commandOK(ctx, "uuid_displace "+commandWords(callID, "start", filePath, "0", "mux")); err != nil {
		return err
	}
	if err := c.SetVariable(ctx, callID, playbackPathVariable, filePath); err != nil {
		cleanupErr := c.stopDisplace(ctx, callID, filePath)
		return errors.Join(err, cleanupErr)
	}
	return nil
}

// StopAudio removes only the media bug created by PlayAudio. The path is kept
// on the FreeSWITCH channel so any API process can stop playback after restart.
func (c *Client) StopAudio(ctx context.Context, callID string) error {
	callID, err := requiredArgument("call ID", callID)
	if err != nil {
		return err
	}
	filePath, err := c.GetVariable(ctx, callID, playbackPathVariable)
	if err != nil {
		return err
	}
	if filePath == "" || filePath == "_undef_" {
		return nil
	}
	if err := c.stopDisplace(ctx, callID, filePath); err != nil {
		return err
	}
	return c.UnsetVariable(ctx, callID, playbackPathVariable)
}

func (c *Client) stopDisplace(ctx context.Context, callID, filePath string) error {
	reply, err := c.Command(ctx, "uuid_displace "+commandWords(callID, "stop", filePath))
	if err != nil {
		return err
	}
	if strings.Contains(strings.ToLower(reply.Body), "cannot stop displace session") {
		return nil
	}
	return commandReplyError(reply)
}

// SendDTMF sends DTMF digits to a call.
func (c *Client) SendDTMF(ctx context.Context, callID, digits string) error {
	callID, err := requiredArgument("call ID", callID)
	if err != nil {
		return err
	}
	digits, err = requiredArgument("DTMF digits", digits)
	if err != nil {
		return err
	}
	return c.commandOK(ctx, "uuid_send_dtmf "+commandWords(callID, digits))
}

func (c *Client) Break(ctx context.Context, callID string) error {
	callID, err := requiredArgument("call ID", callID)
	if err != nil {
		return err
	}
	// uuid_break can wait for an active broadcast application to unwind before
	// its synchronous API response is written. Queue it through bgapi so a media
	// stop cannot monopolize the single request/response ESL connection.
	_, err = c.BGAPI(ctx, "uuid_break "+commandWord(callID))
	return err
}

func (c *Client) Transfer(ctx context.Context, req TransferRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	args := []string{req.CallID, req.Destination}
	if req.Dialplan != "" {
		args = append(args, req.Dialplan)
	}
	if req.Context != "" {
		args = append(args, req.Context)
	}
	return c.commandOK(ctx, "uuid_transfer "+commandWords(args...))
}

func (c *Client) Record(ctx context.Context, req RecordRequest) error {
	if err := req.Validate(); err != nil {
		return err
	}

	action := req.Action
	if action == "" {
		action = "start"
	}
	return c.commandOK(ctx, "uuid_record "+commandWords(req.CallID, action, req.Path))
}
