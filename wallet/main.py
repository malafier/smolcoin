import json
from dataclasses import fields

from prompt_toolkit import prompt
from prompt_toolkit.shortcuts import (
    checkboxlist_dialog,
    input_dialog,
    message_dialog,
    radiolist_dialog,
)

from src.crypto import decrypt_data, encrypt_data, generate_keys, sign_message
from src.network import send_message, send_transaction
from src.storage import load_from_file, save_to_file
from src.transactions import TransactionMessage

### Constants
KEY_STORAGE_PATH = "./key_store.json"


### Wallet State
keys: set[tuple[str, str]] = set()
node_adr: str | None = None


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
            ("broadcast", "Brodcast message"),
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

        elif choice == "broadcast":
            if node_adr:
                send_message(node_adr, "ala ma kota")
            else:
                message_dialog(title="Error", text="Node address is not set.").run()

        elif choice == "transaction":
            if not node_adr:
                message_dialog(title="Error", text="Node address is not set.").run()
                continue
            message = input_dialog(text="Message to send to block").run()
            if not message:
                continue
            key_choice = radiolist_dialog(title="Delete keys", values=key_options).run()
            signed_message = sign_message(key_choice[0], message).decode("utf-8")
            send_transaction(
                node_adr,
                TransactionMessage(message, signed_message, key_choice[0]),
            )

        elif choice == "exit":
            break


if __name__ == "__main__":
    main()
