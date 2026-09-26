# Realtime media v1 acceptance test

This suite builds the pinned `mod_audio_fork`, starts FreeSWITCH and the Go
media worker, creates a short-lived authenticated media session, and forks a
generated tone through the echo engine. It passes only when FreeSWITCH reports
both outbound and returned playback bytes.

Run it from the repository root:

```sh
tests/media-v1/run.sh
```
