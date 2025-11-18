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
    transaction: str
    hash: str
    sender_pk: str
    # r: int
    # s: int
