import json

from prompt_toolkit import prompt
from prompt_toolkit.shortcuts import (
    checkboxlist_dialog,
    input_dialog,
    message_dialog,
    radiolist_dialog,
)

from src.crypto import decrypt_data, encrypt_data, generate_keys
from src.network import broadcast_message
from src.storage import load_from_file, save_to_file

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
        # TODO: dodać # hasła do pliku, tak aby mieć pewność że nie ma ataku na padding
        decrypted = [decrypt_data(password, x) for x in loaded_data]
        for key_pair in decrypted:
            keys.add(tuple(json.loads(key_pair)))
    except:
        print("Failed to decrypt file.")
        exit(1)

    while True:
        radio_options = [
            ("add", "Generate new pair"),
            ("list", "List all pairs"),
            ("delete", "Delete selected pairs"),
            ("connect", "Connect to node"),
            ("broadcast", "Brodcast message"),
            ("exit", "Exit"),
        ]

        choice = radiolist_dialog(title="Wallet", values=radio_options).run()

        if choice == "add":
            add_new_keys(password)
        elif choice == "list":
            displayed_text = ""
            for pair in keys:
                displayed_text += f"{pair[0]}\n{pair[1]}\n\n"
            message_dialog(title="Keys", text=displayed_text).run()
        elif choice == "delete":
            delete_choice = checkboxlist_dialog(
                title="Delete keys", values=[(pair, pair[1]) for pair in keys]
            ).run()
            for pair in delete_choice:
                delete_key(password, pair)
        elif choice == "connect":
            node_adr = input_dialog(text="Provide host address").run()
        elif choice == "broadcast":
            if node_adr:
                broadcast_message(node_adr, "ala ma kota")
            else:
                message_dialog(title="Error", text="Node address is not set.").run()
        elif choice == "exit":
            break


if __name__ == "__main__":
    main()
