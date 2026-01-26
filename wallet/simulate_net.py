import json
import sys
import time

from tqdm import tqdm

import src.crypto as crypto
import src.network as net
from src.storage import load_from_file, save_to_file
from src.transactions import Transaction

TX_URI = "transaction"
URL_TEMPLATE = "http://localhost:%d/%s"
ALA_PTH = "ala_store.json"
BOB_PTH = "bob_store.json"

port = 0


def send_tx(sk: str, _from: str, _to: str):
    tx = Transaction()
    tx.sender = _from
    tx.receiver = _to
    tx.timestamp = int(time.time())
    tx.sign(sk)
    net.send_transaction(f"localhost:{port}", tx)


def register_in_net(pk: str):
    resp = net.get_ledger(f"localhost:{port}", pk)
    print("registered: ", resp)


def main():
    global port
    port = int(sys.argv[1]) if len(sys.argv) > 1 else sys.exit(-1)
    new = sys.argv[2] if len(sys.argv) > 2 else None
    ala = {"login": "ala", "passwd": "ala"}
    bob = {"login": "bob", "passwd": "bob"}

    ala_keys = tuple(
        json.loads(crypto.decrypt_data(ala["passwd"], load_from_file(ALA_PTH)[0]))
        if new
        else crypto.generate_keys()
    )
    bob_keys = tuple(
        json.loads(crypto.decrypt_data(bob["passwd"], load_from_file(BOB_PTH)[0]))
        if new
        else crypto.generate_keys()
    )

    register_in_net(ala_keys[1])
    register_in_net(bob_keys[1])

    for _ in tqdm(range(20)):
        send_tx(ala_keys[0], ala_keys[1], bob_keys[1])
        time.sleep(1)
    print("ala sent her txs")

    for _ in tqdm(range(10)):
        send_tx(bob_keys[0], bob_keys[1], ala_keys[1])
        time.sleep(1)
    print("bob sent his txs")

    ledger = net.get_ledger(f"localhost:{port}", ala_keys[1])
    print("ledger at end:", ledger)

    if new is None:
        encrypted_ala = crypto.encrypt_data(ala["passwd"], json.dumps(ala_keys))
        encrypted_bob = crypto.encrypt_data(bob["passwd"], json.dumps(bob_keys))
        save_to_file(ALA_PTH, [encrypted_ala])
        save_to_file(BOB_PTH, [encrypted_bob])


if __name__ == "__main__":
    main()
