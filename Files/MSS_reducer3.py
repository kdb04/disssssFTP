#!/usr/bin/env python3
import sys

def main():
    current_user = None
    correct_predictions = 0
    incorrect_predictions = 0
    total_value = 0

    for raw_data in sys.stdin:
        user_id, correct, incorrect, api_path, value = raw_data.strip().split()
        
        if current_user is not None and user_id != current_user:
            output_user_summary(current_user, correct_predictions, incorrect_predictions, total_value)
            correct_predictions = 0
            incorrect_predictions = 0
            total_value = 0

        current_user = user_id
        correct_predictions += int(correct)
        incorrect_predictions += int(incorrect)
        total_value += int(value)

    if current_user is not None:
        output_user_summary(current_user, correct_predictions, incorrect_predictions, total_value)

def output_user_summary(user_id, correct, incorrect, value):
    total_predictions = correct + incorrect
    print(f'{user_id} {correct}/{total_predictions} {value}')

if __name__ == "__main__":
    main()