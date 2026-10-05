import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[2]


class EntrypointTest(unittest.TestCase):
    def render(self, advertised):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            config = path / "opensips.cfg"
            config.write_text("socket = udp:0.0.0.0:5060\nsocket = tls:0.0.0.0:5061\nsocket = wss:0.0.0.0:5062\nsocket = tcp:10.1.0.2:5070\n")
            commands = {
                "ip": "if [ \"$2\" = route ]; then echo '1.1.1.1 via 172.30.0.1 dev eth1 src 172.30.0.2'; else printf '1: eth0 inet 172.29.0.2/24 scope global\\n2: eth1 inet 172.30.0.2/24 scope global\\n'; fi",
                "cp": "exit 0",
                "opensips": "exit 0",
            }
            for name, body in commands.items():
                executable = path / name
                executable.write_text("#!/bin/sh\n" + body + "\n")
                executable.chmod(0o755)
            env = dict(os.environ, PATH=str(path) + ":" + os.environ["PATH"],
                       OPENSIPS_CONFIG=str(config), OPENSIPS_DATABASE_URL="postgres://test",
                       OPENSIPS_ADVERTISED_ADDRESS=advertised)
            command = ["sh", str(ROOT / "containers/opensips/entrypoint.sh"), "true"]
            subprocess.run(command, env=env, check=True, capture_output=True)
            first = config.read_text()
            subprocess.run(command, env=env, check=True, capture_output=True)
            self.assertEqual(first, config.read_text())
            return first

    def test_public_identity_preserves_internal_sockets_and_ports(self):
        config = self.render("203.0.113.5")
        self.assertIn("socket = udp:172.30.0.2:5060 as 203.0.113.5:5060", config)
        self.assertIn("socket = tls:172.30.0.2:5061 as 203.0.113.5:5061", config)
        self.assertIn("socket = wss:172.30.0.2:5062 as 203.0.113.5:5062", config)
        self.assertIn("socket = udp:172.29.0.2:5060\n", config)
        self.assertIn("socket = tcp:10.1.0.2:5070\n", config)
        self.assertNotIn("0.0.0.0", config)

    def test_direct_network_has_no_advertised_override(self):
        self.assertNotIn(" as ", self.render(""))

    def test_invalid_advertised_address_fails(self):
        with self.assertRaises(subprocess.CalledProcessError):
            self.render('bad"address')


if __name__ == "__main__":
    unittest.main()
