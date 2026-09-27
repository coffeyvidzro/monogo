package smpp

// Session lifecycle is owned by Client.Bind and Client.Close. Keeping it in
// the adapter prevents SMPP connection state from entering the message domain.
