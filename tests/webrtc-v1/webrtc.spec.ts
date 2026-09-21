import { expect, test } from "@playwright/test";

const required = (name: string): string => {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is required`);
  return value;
};

const requiredPort = (name: string): number => {
  const value = Number(required(name));
  if (!Number.isInteger(value) || value < 1 || value > 65535) throw new Error(`${name} must be a valid port`);
  return value;
};

const unwrap = (payload: any): any => payload?.success === true && "data" in payload ? payload.data : payload;

test("browser call uses a forced TURN relay through RTPengine", async ({ page, request }) => {
  const apiURL = required("LEAMOUT_API_URL");
  const token = required("LEAMOUT_API_TOKEN");
  const domainName = process.env.LEAMOUT_SIP_DOMAIN ?? "webrtc-v1.local";
  const username = process.env.LEAMOUT_SIP_USERNAME ?? "browser";
  const password = process.env.LEAMOUT_SIP_PASSWORD ?? "webrtc-v1-browser-secret";
  const turnRelayMinPort = requiredPort("WEBRTC_V1_TURN_MIN_PORT");
  const turnRelayMaxPort = requiredPort("WEBRTC_V1_TURN_MAX_PORT");
  if (turnRelayMinPort > turnRelayMaxPort) throw new Error("WEBRTC_V1_TURN_MIN_PORT must be <= WEBRTC_V1_TURN_MAX_PORT");

  const headers = { Authorization: `Bearer ${token}` };

  const domainResponse = await request.post(`${apiURL}/v1/sip-domains/`, {
    headers,
    data: { domain: domainName },
  });
  expect(domainResponse.ok(), await domainResponse.text()).toBeTruthy();
  const domain = unwrap(await domainResponse.json());

  const subscriberResponse = await request.post(`${apiURL}/v1/subscribers/`, {
    headers,
    data: {
      sip_domain_id: domain.id,
      username,
      password,
      display_name: "WebRTC v1 acceptance",
    },
  });
  expect(subscriberResponse.ok(), await subscriberResponse.text()).toBeTruthy();
  const subscriber = unwrap(await subscriberResponse.json());

  try {
    const iceResponse = await request.post(`${apiURL}/v1/realtime/ice-credentials`, { headers });
    expect(iceResponse.ok(), await iceResponse.text()).toBeTruthy();
    const credentials = unwrap(await iceResponse.json());
    expect(credentials.ice_servers?.length).toBeGreaterThan(0);

    await page.goto("/");
    await page.evaluate(async ({ credentials, environment }) => {
      await window.runLeamoutWebRTCAcceptance({
        websocketUrl: environment.websocketUrl,
        sipUri: environment.sipUri,
        authorizationUsername: environment.username,
        authorizationPassword: environment.password,
        destinationUri: environment.destinationUri,
        iceServers: credentials.ice_servers,
        turnRelayMinPort: environment.turnRelayMinPort,
        turnRelayMaxPort: environment.turnRelayMaxPort,
      });
    }, {
      credentials,
      environment: {
        websocketUrl: process.env.LEAMOUT_WSS_URL ?? "wss://127.0.0.1:5062",
        sipUri: `sip:${username}@${domainName}`,
        username,
        password,
        destinationUri: process.env.LEAMOUT_DESTINATION_URI ?? `sip:9196@${domainName}`,
        turnRelayMinPort,
        turnRelayMaxPort,
      },
    });
  } finally {
    await request.delete(`${apiURL}/v1/subscribers/${subscriber.id}`, { headers }).catch(() => undefined);
    await request.delete(`${apiURL}/v1/sip-domains/${domain.id}`, { headers }).catch(() => undefined);
  }
});
