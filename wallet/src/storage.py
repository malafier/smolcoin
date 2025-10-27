import json

# import os


def load_from_file(filepath: str) -> list:
    with open(filepath, "r") as file:
        data = json.load(file)
        return data


def save_to_file(filepath: str, data: list):
    with open(filepath, "w") as file:
        json.dump(data, file, ensure_ascii=False)
