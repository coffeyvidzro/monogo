import { Inviter, Registerer, SessionState, UserAgent } from "sip.js";

type AcceptanceConfig = {
  websocketUrl: string;
  sipUri: string;
  authorizationUsername: string;
  authorizationPassword: string;
  destinationUri: string;
  iceServers: RTCIceServer[];
  turnRelayMinPort: number;
  turnRelayMaxPort: number;
};

declare global {
  interface Window {
    runLeamoutWebRTCAcceptance(config: AcceptanceConfig): Promise<void>;
  }
}

const inRelayRange = (port: unknown, minPort: number, maxPort: number): boolean =>
  typeof port === "number" && Number.isInteger(port) && port >= minPort && port <= maxPort;

window.runLeamoutWebRTCAcceptance = async (config) => {
  const uri = UserAgent.makeURI(config.sipUri);
  const destination = UserAgent.makeURI(config.destinationUri);
  if (!uri || !destination) throw new Error("invalid SIP URI");
  if (!Number.isInteger(config.turnRelayMinPort) || !Number.isInteger(config.turnRelayMaxPort)) {
    throw new Error("invalid TURN relay port range");
  }
  if (config.turnRelayMinPort < 1 || config.turnRelayMaxPort > 65535 || config.turnRelayMinPort > config.turnRelayMaxPort) {
    throw new Error("invalid TURN relay port range");
  }

  const userAgent = new UserAgent({
    uri,
    authorizationUsername: config.authorizationUsername,
    authorizationPassword: config.authorizationPassword,
    transportOptions: { server: config.websocketUrl },
    sessionDescriptionHandlerFactoryOptions: {
      peerConnectionConfiguration: {
        iceServers: config.iceServers,
        iceTransportPolicy: "relay",
      },
    },
  });
  const registerer = new Registerer(userAgent);

  try {
    await userAgent.start();
    await registerer.register();

    const inviter = new Inviter(userAgent, destination, {
      sessionDescriptionHandlerOptions: { constraints: { audio: true, video: false } },
    });
    await inviter.invite();
    await new Promise<void>((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error("SIP call did not establish")), 30_000);
      inviter.stateChange.addListener((state) => {
        if (state === SessionState.Established) {
          clearTimeout(timer);
          resolve();
        } else if (state === SessionState.Terminated) {
          clearTimeout(timer);
          reject(new Error("SIP call terminated before establishment"));
        }
      });
    });

    const handler = inviter.sessionDescriptionHandler as unknown as { peerConnection: RTCPeerConnection };
    const peerConnection = handler.peerConnection;
    const remote = document.querySelector<HTMLAudioElement>("#remote");
    if (!remote) throw new Error("remote audio element unavailable");
    remote.srcObject = new MediaStream(peerConnection.getReceivers().flatMap((receiver) => receiver.track ? [receiver.track] : []));

    await new Promise((resolve) => setTimeout(resolve, 2_000));
    const stats = await peerConnection.getStats();
    const selectedPairs = [...stats.values()].filter((report) =>
      report.type === "candidate-pair" && report.state === "succeeded" && report.nominated,
    );
    if (selectedPairs.length !== 1) throw new Error(`expected one selected ICE pair, got ${selectedPairs.length}`);

    const localCandidates = [...stats.values()].filter((report) => report.type === "local-candidate");
    const gatheredRelayCandidates = localCandidates.filter((candidate) =>
      candidate.candidateType === "relay" && inRelayRange(candidate.port, config.turnRelayMinPort, config.turnRelayMaxPort),
    );
    if (gatheredRelayCandidates.length === 0) {
      const observed = localCandidates.map((candidate) => `${candidate.candidateType ?? "unknown"}:${candidate.port ?? "unknown"}`).join(", ");
      throw new Error(`no TURN relay candidate gathered in ${config.turnRelayMinPort}-${config.turnRelayMaxPort}; observed ${observed || "none"}`);
    }

    const localCandidate = stats.get(selectedPairs[0].localCandidateId);
    if (!localCandidate) throw new Error("selected ICE pair has no local candidate stats");
    if (!inRelayRange(localCandidate.port, config.turnRelayMinPort, config.turnRelayMaxPort)) {
      throw new Error(
        `selected ICE candidate escaped TURN relay range ${config.turnRelayMinPort}-${config.turnRelayMaxPort}: ` +
        `${localCandidate.candidateType ?? "unknown"}:${localCandidate.port ?? "unknown"}`,
      );
    }
    if (localCandidate.candidateType !== "relay" && localCandidate.candidateType !== "prflx") {
      throw new Error(`expected selected TURN relay/prflx candidate, got ${localCandidate.candidateType ?? "none"}`);
    }

    const inboundAudio = [...stats.values()].find((report) =>
      report.type === "inbound-rtp" && report.kind === "audio" && report.bytesReceived > 0,
    );
    if (!inboundAudio) throw new Error("no inbound audio received through TURN/RTPengine");
    await inviter.bye();
  } finally {
    await registerer.unregister().catch(() => undefined);
    await userAgent.stop();
  }
};
