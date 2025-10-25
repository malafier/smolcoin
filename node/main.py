import requests
from flask import Flask, jsonify, request


class Node:
    def __init__(self, host, port):
        self.host = host
        self.port = port

        self.peers = set()  # Peers are stored as (host, port) tuples
        self.app = Flask(__name__)
        self.register_routes()

    def register_routes(self):
        @self.app.route("/")
        def index():
            return f"Node running on {self.host}:{self.port}"

        @self.app.route("/peer", methods=["POST"])
        def add_peer():
            """
            Endpoint to add a new peer to the node's peer list.
            Expects JSON data: {"host": "...", "port": ...}
            """
            data = request.get_json()
            if not data or "host" not in data or "port" not in data:
                return (
                    jsonify({"error": "Invalid data. 'host' and 'port' are required."}),
                    400,
                )

            peer_host = data["host"]
            peer_port = int(data["port"])

            # Don't add self
            if peer_host == self.host and peer_port == self.port:
                return jsonify({"message": "Cannot add self as peer."}), 200

            peer_address = (peer_host, peer_port)
            self.peers.add(peer_address)

            print(f"Added new peer: {peer_host}:{peer_port}")
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
            sender = data.get("sender", "Unknown")  # Optionally, see who sent it

            # In a real app, you'd process this message (e.g., add to a blockchain, etc.)
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
                requests.post(url + "/message", json=payload, timeout=2)
                print(f"  - Sent to {peer_host}:{peer_port}")
            except requests.exceptions.RequestException as e:
                print(f"  - Failed to send to {peer_host}:{peer_port}. Error: {e}")
                try:
                    requests.get(url)
                except:
                    print(
                        f"  - Failed to send to connect to {peer_host}:{peer_port}. Peer removed. Error: {e}"
                    )
                    self.peers.remove((peer_host, peer_port))

    def run(self):
        print(f"Starting node on http://{self.host}:{self.port}")
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
    args = parser.parse_args()

    node = Node(args.host, args.port)
    node.run()


if __name__ == "__main__":
    main()
