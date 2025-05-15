import json
import sys

for f in sys.argv[1:]:
    print("convert", f)
    with open(f, 'r') as file:
        data = json.load(file)

    for transaction in data['transaction']:
        transaction['signature'] = {
            'r': transaction.pop('r'),
            's': transaction.pop('s'),
            'y_parity': transaction.pop('y_parity')
        }
        if "authorization_list" in transaction:
            for auth in transaction["authorization_list"]:
                auth['inner'] = {
                    'chain_id': auth.pop('chain_id'),
                    'address': auth.pop('address'),
                    'nonce': auth.pop('nonce')
                }


    with open(f, 'w') as file:
        json.dump(data, file, indent=2)

