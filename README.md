# Raspberry API

API REST para consulta dos indicados e vencedores da categoria **Pior Filme** do Golden Raspberry Awards.

Ao iniciar, a aplicação lê `data/movielist.csv` (separador `;`) e carrega os dados em um
**SQLite em memória** (driver `modernc.org/sqlite`, Go puro). Nenhuma instalação externa é necessária.

## Requisitos
- Go 1.27 ou superior

O arquivo `go.sum` é gerado por `go mod tidy` e deve ser versionado junto com o `go.mod`
(o build do Docker depende dele).

## Como rodar a aplicação
    go mod tidy
    go run ./cmd/api

Variáveis de ambiente opcionais: `ADDR` (padrão `:8080`) e `CSV_PATH` (padrão `data/movielist.csv`).

Com Docker:

    docker build -t raspberry-api .
    docker run -p 8080:8080 raspberry-api

## Como rodar os testes de integração
    go test ./tests/... -v -count=1

Os testes sobem a aplicação completa (banco em memória, carga do CSV e servidor HTTP).
Não há testes unitários, conforme o requisito do projeto. Eles cobrem:

- **CSV oficial** (`data/movielist.csv`): valores esperados fixos.
- **Cenários variados** (`tests/scenarios_test.go`): sem vencedores, sem repetição de prêmios,
  empates em min/max, 3+ prêmios, indicados ignorados, linhas fora de ordem, vários produtores
  por filme, mesmo ano, produtor repetido, formatos de arquivo (BOM, CRLF, colunas em outra ordem).
- **Dados inválidos**: a aplicação não inicia e informa a linha/coluna com problema.
- **Oráculo independente**: 150 datasets gerados aleatoriamente (semente fixa, reproduzível) são
  carregados e a resposta da API é comparada com uma implementação separada da regra.

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

### Regras de negócio
- Só filmes vencedores (`winner = yes`) contam; indicados são ignorados.
- O campo `producers` é separado por vírgula e/ou " and " (sem diferenciar maiúsculas), com espaços
  normalizados. Produtor repetido no mesmo filme conta uma vez.
- O produtor precisa de 2+ prêmios. Intervalos são calculados entre anos **consecutivos** (ordenados).
- Duas vitórias no mesmo ano resultam em intervalo `0`.
- Empates no menor/maior intervalo retornam todos os empatados; sem nenhum caso, `{"min":[],"max":[]}`.
- Ordenação da resposta: `previousWin`, `producer`, `followingWin`.

### Formato do CSV
Separador `;`, cabeçalho obrigatório com as colunas `year`, `title`, `studios`, `producers`, `winner`
(qualquer ordem, qualquer caixa). `winner` aceita `yes`, `no` ou vazio. Aceita BOM UTF-8, CRLF e
linhas em branco. Valor inválido (ano não numérico, `winner` desconhecido, coluna ausente) impede
a inicialização com mensagem de erro indicando a linha.

## Estrutura
    cmd/api               ponto de entrada
    internal/app          montagem da aplicação (banco + carga + HTTP)
    internal/loader       leitura do CSV
    internal/repository   acesso ao SQLite em memória
    internal/service      regra de intervalos de prêmios
    internal/httpapi      handlers HTTP
    tests                 testes de integração
    data                  movielist.csv
