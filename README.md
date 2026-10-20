# stress-test

CLI em Go para testes de carga em serviços web. Dispara um número total de requisições HTTP `GET` contra uma URL, respeitando um nível de concorrência, e ao final imprime um relatório com o resumo da execução.

## Parâmetros

| Flag            | Descrição                                      | Obrigatório |
|-----------------|------------------------------------------------|-------------|
| `--url`         | URL do serviço a ser testado (`http://` ou `https://`) | sim |
| `--requests`    | Número total de requests a realizar (> 0)      | sim         |
| `--concurrency` | Número de chamadas simultâneas (> 0)           | sim         |

Se `--concurrency` for maior que `--requests`, o valor é reduzido para `--requests`. O número total de requests executados é sempre exatamente o informado (a menos que o teste seja interrompido com Ctrl+C, caso em que o relatório mostra o que foi realizado até ali).

## Relatório

Ao final, o console exibe:

- tempo total gasto na execução;
- quantidade total de requests realizados;
- quantidade de requests com status HTTP 200;
- distribuição dos demais códigos de status HTTP (404, 500, ...);
- quantidade de falhas de conexão/timeout (sem resposta HTTP), quando houver.

Exemplo:

```
Testando http://google.com: 1000 requests, 10 simultâneas...

========== Relatório do teste de carga ==========
Tempo total:               9.412s
Total de requests:         1000
Requests por segundo:      106.25
Requests com status 200:   1000
```

## Executando com Docker

### Build da imagem

```bash
docker build -t stress-test .
```

### Execução

```bash
docker run --rm stress-test --url=http://google.com --requests=1000 --concurrency=10
```

> **Atenção:** dentro do container, `localhost` refere-se ao próprio container. Para testar um serviço rodando na sua máquina, use `--network host` (Linux) ou `http://host.docker.internal:<porta>` (Docker Desktop):
>
> ```bash
> docker run --rm --network host stress-test --url=http://localhost:8080 --requests=500 --concurrency=20
> ```

## Executando localmente (sem Docker)

Requer Go 1.26+.

```bash
go run . --url=http://google.com --requests=1000 --concurrency=10
# ou
go build -o stress-test . && ./stress-test --url=http://google.com --requests=1000 --concurrency=10
```

## Testes

```bash
go test -race ./...
```

## Estrutura

```
main.go                       ponto de entrada
cmd/root.go                   comando cobra, flags e validação
internal/loadtest/runner.go   pool de workers e execução das requisições
internal/loadtest/report.go   estrutura e impressão do relatório
Dockerfile                    build multi-stage (imagem final `scratch`)
```
