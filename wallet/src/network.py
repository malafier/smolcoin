import dataclasses
import json

import requests

from src.transactions import TransactionMessage


def send_message(address: str, message: str):
    requests.post(url=f"http://{address}/broadcast", json={"message": message})


def send_transaction(address: str, transaction: TransactionMessage):
    requests.post(
        url=f"http://{address}/transaction",
        json=json.dumps(dataclasses.asdict(transaction)),
    )
