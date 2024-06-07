# OMS (Order Management System)

## Local

### Docker Compose

For external services like RabbitMQ and JaggerUI, you can use docker compose to start them up.
```bash
cd ..
docker compose up --build
```

### Start the services

```bash
cd order && air
cd payment && air
...
```

### Start Stripe Server

Run the following command to start the stripe cli
```bash
stripe login
```

Then run the following command to listen for webhooks

```bash
 stripe listen --forward-to localhost:8081/webhook
```

Where `localhost:8081/webhook` is the endpoint `payment service` HTTP server address.

Test card: 4242424242424242


## RabbitMQ UI

http://localhost:15672/#/

## Jaeger UI


## Deployment

Build Docker images for each microservice and push them to a container registry.
Deploy using Docker Compose or orchestration tools like Kubernetes.

Publishable key
pk_test_51POf3ARwn3euj82DM78K8M066Bx9KcmH7km59i3bdpYS90TkoPedyvik0PBbG4WtZtoa7OiZV7bupAi7TNpzOAkp00UuNKwOif

Secret key
sk_test_51POf3ARwn3euj82DTKV1gVEqsvjI1p51zQVN5MwmLhcdRh7pfWhU1G0ADka6aK1x5S8DmYaeREEdaGAcESyn7Q9o00d7jecplw