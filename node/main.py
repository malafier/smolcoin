import requests
from flask import Flask, jsonify, request

app = Flask(__name__)

known_peers = set()
messages = []


@app.route("/message", methods=["POST"])
def receive_message():
    data = request.get_json()
    msg = data.get("message")
    sender = data.get("sender")

    if msg:
        messages.append((sender, msg))
        print(f"📩 Message from {sender}: {msg}")
    return jsonify({"status": "ok"}), 200


@app.route("/send", methods=["POST"])
def send_message():
    data = request.get_json()
    msg = data.get("message")
    if not msg:
        return jsonify({"error": "No message provided"}), 400

    for peer in known_peers:
        try:
            requests.post(
                f"http://{peer}/message", json={"sender": request.host, "message": msg}
            )
        except Exception as e:
            print(f"⚠️ Could not send to {peer}: {e}")

    return jsonify({"status": "sent", "message": msg}), 200


@app.route("/peers", methods=["GET", "POST"])
def peers():
    if request.method == "POST":
        data = request.get_json()
        peer = data.get("peer")
        if peer:
            known_peers.add(peer)
        return jsonify({"peers": list(known_peers)})
    else:
        return jsonify({"peers": list(known_peers)})


@app.route("/")
def home():
    return jsonify({"known_peers": list(known_peers), "messages": messages})


def connect_to_new_peers():
    while len(known_peers) < 3:
        for peer in known_peers:
            new_peers = requests.get(f"http://{peer}/peers")
            for np in new_peers:
                known_peers.add(np)


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--port", type=int, required=True, help="Port number to run the peer"
    )
    parser.add_argument(
        "--connect", type=str, help="Address of another peer (host:port)"
    )
    args = parser.parse_args()

    if args.connect:
        known_peers.add(args.connect)
        try:
            requests.post(
                f"http://{args.connect}/peers", json={"peer": f"127.0.0.1:{args.port}"}
            )
        except:
            print(f"⚠️ Could not connect to {args.connect}")

    app.run(host="0.0.0.0", port=args.port)
