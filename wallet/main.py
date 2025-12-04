import base64
import json

import requests
from prompt_toolkit import prompt
from prompt_toolkit.shortcuts import (
    checkboxlist_dialog,
    input_dialog,
    message_dialog,
    radiolist_dialog,
)

from src.crypto import decrypt_data, encrypt_data, generate_keys, sign_message
from src.network import get_possible_pubs, send_transaction
from src.storage import load_from_file, save_to_file
from src.transactions import Transaction, TransactionMessage

### Constants
KEY_STORAGE_PATH = "./key_store.json"


### Wallet State
keys: set[tuple[str, str]] = set()
node_adr: str | None = "localhost:3001"


def save_keys(password: str):
    data = []
    for pair in keys:
        encrypted = encrypt_data(password, json.dumps(pair))
        data.append(encrypted)
    save_to_file(KEY_STORAGE_PATH, data)


def add_new_keys(password: str):
    pair = generate_keys()
    keys.add(pair)
    save_keys(password)


def delete_key(password: str, pair: tuple[str, str]):
    keys.remove(pair)
    save_keys(password)


def main():
    global node_adr
    password = prompt("Enter password: ", is_password=True)

    loaded_data = load_from_file(KEY_STORAGE_PATH)
    try:
        decrypted = [decrypt_data(password, x) for x in loaded_data]
        for key_pair in decrypted:
            keys.add(tuple(json.loads(key_pair)))
    except:
        print("Failed to decrypt file.")
        exit(1)

    while True:
        key_options = [(pair, pair[1]) for pair in keys]
        radio_options = [
            ("add", "Generate new pair"),
            ("list", "List all pairs"),
            ("delete", "Delete selected pairs"),
            ("connect", "Connect to node"),
            ("transaction", "Send transaction"),
            ("exit", "Exit"),
        ]
        text = f"Connected to node: {node_adr}"
        choice = radiolist_dialog(title="Wallet", text=text, values=radio_options).run()

        if choice == "add":
            add_new_keys(password)

        elif choice == "list":
            displayed_text = ""
            for pair in keys:
                displayed_text += f"{pair[0]}\n{pair[1]}\n\n"
            message_dialog(title="Keys", text=displayed_text).run()

        elif choice == "delete":
            delete_choice = checkboxlist_dialog(
                title="Delete keys", values=key_options
            ).run()
            for pair in delete_choice:
                delete_key(password, pair)

        elif choice == "connect":
            node_adr = input_dialog(text="Provide node address").run()
            try:
                requests.get(url=f"http://{node_adr}/")
            except:
                node_adr = None

        elif choice == "transaction":
            if not node_adr:
                message_dialog(title="Error", text="Node address is not set.").run()
                break

            transaction = Transaction()
            key_chosen = None
            while True:
                transaction_options = [
                    (
                        "sender",
                        f"Sender: {key_chosen if not key_chosen else key_chosen[1][:8]}",
                    ),
                    ("reciever", f"Reciever: {transaction.reciever}"),
                    ("ammount", f"Ammount: {transaction.ammount}"),
                    ("difficulty", f"Difficulty: {transaction.difficulty}"),
                    ("send", "Send"),
                ]

                transaction_choice = radiolist_dialog(
                    title="New Transaction", values=transaction_options
                ).run()
                if transaction_choice == "sender":
                    key_chosen = radiolist_dialog(
                        title="Chose identity", values=key_options
                    ).run()[1]
                elif transaction_choice == "reciever":
                    reciever_choice = get_possible_pubs(node_adr)
                    transaction.reciever = radiolist_dialog(
                        title="Chose reciever", values=[(x, x) for x in reciever_choice]
                    ).run()
                elif transaction_choice == "ammount":
                    transaction.ammount = float(input_dialog(text="Set ammount").run())
                elif transaction_choice == "difficulty":
                    transaction.difficulty = int(
                        input_dialog(text="Set difficulty").run()
                    )
                elif transaction_choice == "send":
                    break

            if not key_chosen:
                break
            transaction.sender = key_chosen[1]
            trans_str = json.dumps(transaction, sort_keys=True)
            signature_str = base64.b64encode(
                sign_message(key_chosen[0], trans_str)
            ).decode("utf-8")
            send_transaction(
                node_adr,
                TransactionMessage(trans_str, signature_str, key_chosen[1]),
            )

        elif choice == "exit":
            break


if __name__ == "__main__":
    main()
