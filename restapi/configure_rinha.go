// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"context"
	"crypto/tls"
	"net/http"
	"os"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rinha/models"
	"rinha/restapi/operations"

	_ "github.com/joho/godotenv/autoload"

	"rinha/db"
)

//go:generate swagger generate server --target ../../rinha2024-q1-go-swagger-sqlc-postgres --name Rinha --spec ../swagger.yml --principal interface{}

func configureFlags(api *operations.RinhaAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.RinhaAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	ctx := context.Background()
	connString := "user=" + os.Getenv("DB_USER") +
		" password=" + os.Getenv("DB_PASSWORD") +
		" dbname=" + os.Getenv("DB_NAME") +
		" host=" + os.Getenv("DB_HOST") +
		" sslmode=disable"

	conn, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		panic(err)
	}
	queries := db.New(conn)

	api.ConsultarExtratoHandler = operations.ConsultarExtratoHandlerFunc(func(params operations.ConsultarExtratoParams) middleware.Responder {
		extrato, err := queries.Extrato(ctx, int32(params.ID))
		if err != nil {
			if err == pgx.ErrNoRows {
				return operations.NewConsultarExtratoNotFound()
			}
			return operations.NewConsultarExtratoInternalServerError()
		}
		if len(extrato) == 0 {
			return operations.NewConsultarExtratoNotFound()
		}

		saldoextrato := models.SaldoExtrato{
			Total:       &extrato[0].Saldo,
			Limite:      &extrato[0].Limite,
			DataExtrato: (*strfmt.DateTime)(&extrato[0].DataExtrato.Time),
		}
		var ultimastransacoes []*models.ListaExtrato
		for _, v := range extrato {
			if v.TipoOperacao == "e" {
				break
			}
			t := v
			ultimastransacoes = append(ultimastransacoes,
				&models.ListaExtrato{
					Valor:       &t.Valor.Int64,
					Tipo:        &t.TipoOperacao,
					Descricao:   &t.Descricao,
					RealizadaEm: (*strfmt.DateTime)(&t.RealizadaEm.Time),
				})
		}
		return operations.NewConsultarExtratoOK().WithPayload(&models.Extrato{
			Saldo:             &saldoextrato,
			UltimasTransacoes: ultimastransacoes,
		})
	})

	api.RealizarTransacaoHandler = operations.RealizarTransacaoHandlerFunc(func(params operations.RealizarTransacaoParams) middleware.Responder {
		if *params.Body.Tipo == "c" {
			credito, err := queries.Credito(ctx, db.CreditoParams{
				Valor:     *params.Body.Valor,
				Descricao: *params.Body.Descricao,
				IDConta:   int32(params.ID),
			})
			if err != nil {
				if err == pgx.ErrNoRows {
					return operations.NewRealizarTransacaoNotFound()
				}
				return operations.NewRealizarTransacaoInternalServerError()
			}

			return operations.NewRealizarTransacaoOK().WithPayload(&models.TransacaoOutput{
				Limite: &credito.Limite,
				Saldo:  &credito.Saldo,
			})
		}
		if *params.Body.Tipo == "d" {
			debito, err := queries.Debito(ctx, db.DebitoParams{
				Valor:     *params.Body.Valor,
				Descricao: *params.Body.Descricao,
				IDConta:   int32(params.ID),
			})
			if err != nil {
				if err == pgx.ErrNoRows {
					return operations.NewRealizarTransacaoNotFound()
				}
				return operations.NewRealizarTransacaoInternalServerError()
			}

			if !debito.Autorizado {
				return operations.NewRealizarTransacaoUnprocessableEntity()
			}

			return operations.NewRealizarTransacaoOK().WithPayload(&models.TransacaoOutput{
				Limite: &debito.Limite,
				Saldo:  &debito.Saldo,
			})
		}
		return operations.NewRealizarTransacaoInternalServerError()
	})

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {
		conn.Close()
	}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
