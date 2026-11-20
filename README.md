# Raspberry API

API REST para consulta dos indicados e vencedores da categoria **Pior Filme** do Golden Raspberry Awards.
Ao iniciar, a aplicação lê `data/movielist.csv` (separador `;`) e carrega os dados em um
**SQLite em memória** (driver `modernc.org/sqlite`, Go puro). Nenhuma instalação externa é necessária.

## Requisitos
- Go 1.27 ou superior

## Como rodar a aplicação
    go mod tidy
    go run ./cmd/api

Variáveis de ambiente opcionais: `ADDR` (padrão `:8080`) e `CSV_PATH` (padrão `data/movielist.csv`).

Com Docker:

    docker build -t raspberry-api .
    docker run -p 8080:8080 raspberry-api

## Como rodar os testes de integração
    go test ./tests/... -v

Os testes sobem a aplicação completa (banco em memória, carga do CSV e servidor HTTP) e
verificam que as respostas correspondem aos dados do CSV fornecido.
Não há testes unitários, conforme o requisito do projeto.

## Endpoints
| Método | Caminho                | Descrição                                                        |
|--------|------------------------|------------------------------------------------------------------|
| GET    | `/producers/intervals` | Produtores com menor e maior intervalo entre prêmios consecutivos |
| GET    | `/movies`              | Lista filmes (filtros opcionais: `?year=1980&winner=true`)       |
| GET    | `/movies/{id}`         | Busca um filme                                                   |
| GET    | `/healthz`             | Health check                                                     |

Códigos de status: `200` sucesso, `400` parâmetro inválido, `404` recurso inexistente,
`405` método não permitido, `500` erro interno.

### Exemplo: `GET /producers/intervals`
    {
      "min": [{"producer": "Joel Silver", "interval": 1, "previousWin": 1990, "followingWin": 1991}],
      "max": [{"producer": "Matthew Vaughn", "interval": 13, "previousWin": 2002, "followingWin": 2015}]
    }

Regras: considera só vencedores; o campo `producers` é separado por vírgula e/ou " and ";
em caso de empate, todos os produtores empatados são retornados.

## Estrutura
    cmd/api               ponto de entrada
    internal/app          montagem da aplicação (banco + carga + HTTP)
    internal/loader       leitura do CSV
    internal/repository   acesso ao SQLite em memória
    internal/service      regra de intervalos de prêmios
    internal/httpapi      handlers HTTP
    tests                 testes de integração
    data                  movielist.csv
