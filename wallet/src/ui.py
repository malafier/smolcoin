import json
import time

import requests
from prompt_toolkit.shortcuts import (
    checkboxlist_dialog,
    input_dialog,
    message_dialog,
    radiolist_dialog,
)

from src.crypto import encrypt_data, generate_keys
from src.network import get_ledger, get_possible_ids, send_transaction
from src.storage import save_to_file
from src.transactions import Transaction

type KeyPair = tuple[str, str]
PK = 1
SK = 0


def view_key(key: str):
    return (
        key.replace("-----BEGIN PUBLIC KEY-----", "")
        .replace("-----END PUBLIC KEY-----", "")
        .replace("\n", "")
        .strip()
    )


class App:
    def __init__(self, password, keys, storage_path, node_adr=None) -> None:
        self.password: str = password
        self.keys: set[KeyPair] = keys
        self.storage_path = storage_path
        self.node_adr: None | str = node_adr

        self.chosen_id: KeyPair | None = None
        self.coins: float | None = None

    def run(self):
        while True:
            self._update_coins()
            if self._render():
                break

    def _render(self) -> None | str:
        key_options = [(pair, pair[1]) for pair in self.keys]
        radio_options = [
            ("add", "Generate new pair"),
            ("list", "Chose identity"),
            ("delete", "Delete selected pairs"),
            ("connect", "Connect to node"),
            ("transaction", "Send transaction"),
            ("exit", "Exit"),
        ]
        text = f"""Connected to node: {self.node_adr}
Identity: {'Chose identity' if not self.chosen_id else '...'+ view_key(self.chosen_id[PK])[-24:]}
Smolcoins: {'No info' if self.coins is None else self.coins}
        """
        choice = radiolist_dialog(title="Wallet", text=text, values=radio_options).run()

        if choice == "add":
            self._add_new_keys()

        elif choice == "list":
            if len(key_options) == 0:
                return
            self.chosen_id = radiolist_dialog(
                title="Chose identity", values=key_options
            ).run()

        elif choice == "delete":
            delete_choice = checkboxlist_dialog(
                title="Delete keys", values=key_options
            ).run()
            for pair in delete_choice:
                self._delete_key(pair)

        elif choice == "connect":
            node_adr = input_dialog(text="Provide node address").run()
            try:
                requests.get(url=f"http://{node_adr}/")
            except:
                node_adr = None

        elif choice == "transaction":
            if not self.node_adr:
                message_dialog(title="Error", text="Node address is not set.").run()
                return
            if not self.chosen_id:
                message_dialog(title="Error", text="Identity is not chosen").run()
                return

            tx = self._render_tx()
            if not tx:
                return
            tx.sign(self.chosen_id[SK])
            send_transaction(self.node_adr, tx)

        elif choice == "exit":
            return "EXIT"

    def _render_tx(self) -> None | Transaction:
        assert isinstance(self.node_adr, str)
        assert self.chosen_id is not None

        tx = Transaction()
        tx.sender = self.chosen_id[PK]

        while True:
            transaction_options = [
                ("reciever", f"Reciever: {'...' + view_key(tx.reciever)[-24:]}"),
                ("ammount", f"Ammount: {tx.ammount}"),
                ("send", "Send"),
                ("cancel", "Cancel transaction"),
            ]
            tx_choice = radiolist_dialog(
                title="New Transaction",
                values=transaction_options,
                text=f"Sender: {  '...'+view_key(self.chosen_id[PK])[-24:]}",
            ).run()

            if tx_choice == "reciever":
                reciever_choice = get_possible_ids(self.node_adr)
                tx.reciever = radiolist_dialog(
                    title="Chose reciever",
                    values=[(x, x) for x in reciever_choice],
                ).run()
            elif tx_choice == "ammount":
                tx.ammount = float(input_dialog(text="Set ammount").run())
            elif tx_choice == "send":
                break
            elif tx_choice == "cancel":
                return
        tx.timestamp = int(time.time())
        return tx

    def _save_keys(self):
        data = []
        for pair in self.keys:
            encrypted = encrypt_data(self.password, json.dumps(pair))
            data.append(encrypted)
        save_to_file(self.storage_path, data)

    def _add_new_keys(self):
        pair = generate_keys()
        self.keys.add(pair)
        self._save_keys()

    def _delete_key(self, pair: tuple[str, str]):
        self.keys.remove(pair)
        self._save_keys()

    def _update_coins(self):
        if not self.chosen_id or not self.node_adr:
            return
        ledger = get_ledger(self.node_adr, self.chosen_id[PK])
        if ledger is None:
            return
        self.coins = float(ledger[self.chosen_id[PK]])
