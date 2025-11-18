from dataclasses import dataclass


@dataclass
class Transaction:
    sender: str
    reciever: str
    ammount: float
    timestamp: int
    difficulty: int


@dataclass
class TransactionMessage:
    transaction: Transaction
    hash: str
    sender_pk: str
    r: int
    s: int


transaction_fields = [
    {"label": "Sender", "name": "sender"},
    {"label": "Reciever", "name": "reciever"},
    {"label": "Ammount", "name": "ammount"},
]
