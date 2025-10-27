import json

from src.crypto import generate_keys

### Wallet State
keys = set()
password: str | None = None


def main():
    prv, pub = generate_keys()


if __name__ == "__main__":
    main()
