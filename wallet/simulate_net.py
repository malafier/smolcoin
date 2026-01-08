import json
import sys

import src.crypto as crypto
import src.network as net
from src.storage import save_to_file
from src.transactions import Transaction

TX_URI = "transaction"
URL_TEMPLATE = "http://localhost:%d/%s"

port = 0


def send_tx(sk: str, _from: str, _to: str):
    tx = Transaction()
    tx.sender = _from
    tx.reciever = _to
    tx.sign(sk)
    net.send_transaction(f"localhost:{port}", tx)


def register_in_net(pk: str):
    net.get_ledger(f"localhost:{port}", pk)


def main():
    global port
    port = int(sys.argv[1])
    ala = {"login": "ala", "passwd": "ala"}
    bob = {"login": "bob", "passwd": "bob"}

    ala_keys = crypto.generate_keys()
    bob_keys = crypto.generate_keys()

    register_in_net(ala_keys[1])
    register_in_net(bob_keys[1])

    for _ in range(50):
        send_tx(ala_keys[0], ala_keys[1], bob_keys[1])

    for _ in range(20):
        send_tx(bob_keys[0], bob_keys[1], ala_keys[1])

    encrypted_ala = crypto.encrypt_data(ala["passwd"], json.dumps(ala_keys))
    encrypted_bob = crypto.encrypt_data(bob["passwd"], json.dumps(bob_keys))
    save_to_file("ala_storage.json", [encrypted_ala])
    save_to_file("bob_storage.json", [encrypted_bob])


if __name__ == "__main__":
    main()
