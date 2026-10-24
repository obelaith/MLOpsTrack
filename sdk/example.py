
from sdk.mlopstrack import Client

client = Client()

experiment = client.create_experiment(
    "sdk-demo",
    "Testing the MLOpsTrack Python SDK",
)

run = client.start_run(experiment["id"], "baseline")

run.log_param("learning_rate", 0.001)
run.log_param("optimizer", "adamw")
run.log_param("batch_size", 32)

losses = [0.82, 0.54, 0.31]

for step, loss in enumerate(losses, start=1):
    run.log_metric("loss", loss, step)

result = run.finish()

print("Experiment ID:", experiment["id"])
print("Run ID:", run.id)
print("Run status:", result["status"])
