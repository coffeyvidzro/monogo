import { expect, test } from "@playwright/test";

const required = (name: string): string => {
    const value = process.env[name];
    if (!value) throw new Error(`${name} is required`);
    return value;
};

const requiredPort = (name: string): number => {
    const value = Number(required(name));
    if (!Number.isInteger(value) || value < 1 || value > 65535)
        throw new Error(`${name} must be a valid port`);
    return value;
};

const unwrap = (payload: any): any =>
    payload?.success === true && "data" in payload ? payload.data : payload;

test("browser gathers a forced TURN relay candidate", async ({ page, request }) => {
    const apiURL = required("LEAMOUT_API_URL");
    const token = required("LEAMOUT_API_TOKEN");
    const turnRelayMinPort = requiredPort("WEBRTC_V1_TURN_MIN_PORT");
    const turnRelayMaxPort = requiredPort("WEBRTC_V1_TURN_MAX_PORT");

    const response = await request.post(`${apiURL}/v1/webrtc/ice-credentials`, {
        headers: { Authorization: `Bearer ${token}` },
    });
    expect(response.ok(), await response.text()).toBeTruthy();

    const credentials = unwrap(await response.json());
    expect(credentials.ice_servers?.length).toBeGreaterThan(0);

    const result = await page.evaluate(
        async ({ iceServers, minPort, maxPort }) => {
            const pc = new RTCPeerConnection({
                iceServers,
                iceTransportPolicy: "relay",
            });

            try {
                pc.createDataChannel("probe");
                const candidates: Array<{ type: string; port: number | null }> = [];
                pc.onicecandidate = (event) => {
                    if (!event.candidate) return;
                    const parsed = event.candidate.candidate.split(/\s+/);
                    const typ = parsed.indexOf("typ");
                    candidates.push({
                        type: typ >= 0 ? parsed[typ + 1] : "unknown",
                        port: Number(parsed[5]) || null,
                    });
                };

                await pc.setLocalDescription(await pc.createOffer());
                await new Promise<void>((resolve, reject) => {
                    const timer = setTimeout(
                        () => reject(new Error("ICE gathering timed out")),
                        15_000,
                    );
                    const check = () => {
                        if (pc.iceGatheringState === "complete") {
                            clearTimeout(timer);
                            resolve();
                        }
                    };
                    pc.addEventListener("icegatheringstatechange", check);
                    check();
                });

                const relay = candidates.find(
                    (candidate) =>
                        candidate.type === "relay" &&
                        candidate.port !== null &&
                        candidate.port >= minPort &&
                        candidate.port <= maxPort,
                );

                return { candidates, relay };
            } finally {
                pc.close();
            }
        },
        {
            iceServers: credentials.ice_servers,
            minPort: turnRelayMinPort,
            maxPort: turnRelayMaxPort,
        },
    );

    expect(result.relay, JSON.stringify(result.candidates)).toBeTruthy();
});
