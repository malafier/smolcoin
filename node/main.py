import requests
from flask import Flask, jsonify, request


class Node:
    def __init__(self, host, port):
        self.host = host
        self.port = port
        self.peer_header = {"Peer": f"{host}:{port}"}

        self.peers = set()
        self.app = Flask(__name__)
        self.register_routes()

    def _apply_middleware(self):
        @self.app.before_request
        def check_peer_header():
            if len(self.peers) > 3 or not request.headers.get("Peer"):
                return
            peer_info = request.headers.get("Peer")
            assert type(peer_info) == str
            host, port_str = peer_info.split(":")
            port = int(port_str)
            self.peers.add((host, port))

    def register_routes(self):
        self._apply_middleware()

        @self.app.route("/")
        def index():
            return f"Node running on {self.host}:{self.port}", 200

        @self.app.route("/peer", methods=["POST"])
        def add_peer():
            data = request.get_json()
            if not data or "host" not in data or "port" not in data:
                return (
                    jsonify({"error": "Invalid data. 'host' and 'port' are required."}),
                    400,
                )

            peer_host = data["host"]
            peer_port = int(data["port"])

            if peer_host == self.host and peer_port == self.port:
                return jsonify({"message": "Cannot add self as peer."}), 200

            peer_address = (peer_host, peer_port)
            self.peers.add(peer_address)

            print(f"[I] Added new peer: {peer_host}:{peer_port}")
            return (
                jsonify(
                    {"message": "Peer added successfully.", "peers": list(self.peers)}
                ),
                201,
            )

        @self.app.route("/peers", methods=["GET"])
        def get_peers():
            return jsonify({"peers": list(self.peers)}), 200

        @self.app.route("/message", methods=["POST"])
        def receive_message():
            data = request.get_json()
            if not data or "message" not in data:
                return jsonify({"error": "Invalid data. 'message' is required."}), 400

            message = data["message"]
            sender = data.get("sender", "Unknown")

            print(f"\n[Message Received from {sender}]: {message}\n")

            return jsonify({"message": "Message received."}), 200

        @self.app.route("/broadcast", methods=["POST"])
        def broadcast_message_route():
            data = request.get_json()
            if not data or "message" not in data:
                return jsonify({"error": "Invalid data. 'message' is required."}), 400

            self.broadcast_message(data["message"])
            return jsonify({"message": "Broadcast initiated."}), 200

    def broadcast_message(self, message):
        print(f"Broadcasting message to {len(self.peers)} peer(s)...")
        payload = {"message": message, "sender": f"{self.host}:{self.port}"}

        for peer_host, peer_port in list(self.peers):
            url = f"http://{peer_host}:{peer_port}"
            try:
                requests.post(
                    url + "/message", json=payload, headers=self.peer_header, timeout=2
                )
                print(f"  - [I] Sent to {peer_host}:{peer_port}")
            except requests.exceptions.RequestException as e:
                print(f"  - [W] Failed to send to {peer_host}:{peer_port}. Error: {e}")
                try:
                    requests.get(url)
                except:
                    print(
                        f"  - [E] Failed to send to connect to {peer_host}:{peer_port}. Peer removed. Error: {e}"
                    )
                    self.peers.remove((peer_host, peer_port))

    def run(self, initial_peer: None | str = None):
        if initial_peer:
            host, port_str = initial_peer.split(":")
            port = int(port_str)
            url = f"http://{host}:{port}"
            try:
                requests.get(url, headers=self.peer_header)
                self.peers.add((host, port))
            except:
                print(f"[E] Failed to send to connect to {host}:{port}.")

        print(f"[I] Starting node on http://{self.host}:{self.port}")
        self.app.run(host=self.host, port=self.port, debug=False, use_reloader=False)


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Run a P2P node.")
    parser.add_argument(
        "-p",
        "--port",
        type=int,
        default=3000,
        help="The port number for the node to listen on.",
    )
    parser.add_argument(
        "-H",
        "--host",
        type=str,
        default="127.0.0.1",
        help="The host address for the node to bind to (default: 127.0.0.1).",
    )
    parser.add_argument(
        "-P",
        "--peer",
        type=str,
        help="Adress to connect to peer",
    )
    args = parser.parse_args()

    node = Node(args.host, args.port)
    node.run(args.peer)


if __name__ == "__main__":
    main()
