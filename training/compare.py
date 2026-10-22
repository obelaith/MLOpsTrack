
from sdk.mlopstrack import Client
from training.train import train


def main():
    client = Client()

    experiment = client.create_experiment(
        "learning-rate-comparison",
        "Comparing SGD learning rates",
    )

    for learning_rate in [0.01, 0.1, 0.2]:
        train(client, experiment["id"], learning_rate)

    results = []

    for run in client.list_runs(experiment["id"]):
        if run["status"] != "completed":
            continue

        metrics = client.list_metrics(run["id"])

        accuracy = next(
            (
                metric["value"]
                for metric in metrics
                if metric["name"] == "val_accuracy"
                and metric["step"] == 20
            ),
            None,
        )

        if accuracy is None:
            raise RuntimeError(
                f"Run {run['id']} has no final validation accuracy"
            )

        results.append((accuracy, run["name"], run["id"]))

    results.sort(reverse=True)

    print("\nComparison results:")

    for accuracy, name, run_id in results:
        print(f"{name} | Run {run_id} | Accuracy {accuracy:.2%}")

    print(f"\nBest run: {results[0][1]}")


if __name__ == "__main__":
    main()
