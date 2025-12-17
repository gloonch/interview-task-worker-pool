## 1) Create task (POST /tasks)
```
curl -i -X POST "http://localhost:8080/tasks" \
-H "Content-Type: application/json" \
-d '{"title":"Buy groceries","description":"Milk, eggs, bread"}'
```

## 2) Get task by ID (GET /tasks/{id})
```
curl -i "http://localhost:8080/tasks/1"
```

## 3) List tasks (GET /tasks)
```
curl -i "http://localhost:8080/tasks"
```

## invalid json -> 400
```
curl -i -X POST "http://localhost:8080/tasks" \
-H "Content-Type: application/json" \
-d '{bad json}'
```

## task not found -> 404
```
curl -i "http://localhost:8080/tasks/999999"
```
