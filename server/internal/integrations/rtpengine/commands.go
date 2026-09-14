package rtpengine

import "context"

func (c *Client) Offer(ctx context.Context, request OfferRequest) (OfferResponse, error) {
	if err := request.Validate(); err != nil {
		return OfferResponse{}, err
	}

	response, err := c.exchangeSDP(ctx, CommandOffer, request.Session, request.SDP, request.Flags)
	if err != nil {
		return OfferResponse{}, err
	}

	sdp, err := response.RequiredString("sdp")
	if err != nil {
		return OfferResponse{}, err
	}

	return OfferResponse{SDP: sdp, Data: response.Data}, nil
}

func (c *Client) Answer(ctx context.Context, request AnswerRequest) (AnswerResponse, error) {
	if err := request.Validate(); err != nil {
		return AnswerResponse{}, err
	}

	response, err := c.exchangeSDP(ctx, CommandAnswer, request.Session, request.SDP, request.Flags)
	if err != nil {
		return AnswerResponse{}, err
	}

	sdp, err := response.RequiredString("sdp")
	if err != nil {
		return AnswerResponse{}, err
	}

	return AnswerResponse{SDP: sdp, Data: response.Data}, nil
}

func (c *Client) Delete(ctx context.Context, request DeleteRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}

	_, err := c.do(ctx, CommandDelete, sessionParams(request.Session, "", request.Flags))
	return err
}

func (c *Client) Query(ctx context.Context, session Session) (QueryResponse, error) {
	if err := session.Validate(); err != nil {
		return QueryResponse{}, err
	}

	response, err := c.do(ctx, CommandQuery, sessionParams(session, "", nil))
	if err != nil {
		return QueryResponse{}, err
	}

	return QueryResponse{Data: response.Data}, nil
}

func (c *Client) exchangeSDP(
	ctx context.Context,
	command Command,
	session Session,
	sdp string,
	flags []string,
) (Response, error) {
	return c.do(ctx, command, sessionParams(session, sdp, flags))
}
