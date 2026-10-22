
import torch

from sdk.mlopstrack import Client


def train(client, experiment_id, learning_rate=0.2):
    torch.manual_seed(42)

    x = torch.randn(1000, 2)
    y = (x[:, 0] + x[:, 1] > 0).float().unsqueeze(1)

    x_train, x_val = x[:800], x[800:]
    y_train, y_val = y[:800], y[800:]

    model = torch.nn.Linear(2, 1)
    loss_fn = torch.nn.BCEWithLogitsLoss()
    optimizer = torch.optim.SGD(
        model.parameters(), lr=learning_rate
    )

    run = client.start_run(
        experiment_id, f"sgd-lr-{learning_rate:g}"
    )

    try:
        run.log_param("learning_rate", learning_rate)
        run.log_param("epochs", 20)
        run.log_param("optimizer", "SGD")
        run.log_param("seed", 42)

        for epoch in range(1, 21):
            model.train()
            optimizer.zero_grad()

            loss = loss_fn(model(x_train), y_train)
            loss.backward()
            optimizer.step()

            model.eval()
            with torch.no_grad():
                logits = model(x_val)
                val_loss = loss_fn(logits, y_val).item()
                accuracy = (
                    (logits >= 0) == y_val.bool()
                ).float().mean().item()

            run.log_metric("train_loss", loss.item(), epoch)
            run.log_metric("val_loss", val_loss, epoch)
            run.log_metric("val_accuracy", accuracy, epoch)

            print(
                f"LR {learning_rate:g} | Epoch {epoch:02d} | "
                f"Loss {loss.item():.4f} | "
                f"Accuracy {accuracy:.2%}"
            )

        run.finish()

    except Exception:
        try:
            run.finish("failed")
        except Exception as error:
            print(f"Could not mark run failed: {error}")
        raise

    print(f"Completed run {run.id}")
    return run.id


def main():
    client = Client()

    experiment = client.create_experiment(
        "binary-classification",
        "PyTorch classification experiment",
    )

    train(client, experiment["id"])


if __name__ == "__main__":
    main()
