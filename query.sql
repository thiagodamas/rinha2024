-- name: Credito :one
with dados as (
	select * from dados_bancarios 
	where dados_bancarios.id_conta=sqlc.arg(id_conta)
	for update
),
informacoes as (
    select 
	case
	  when (saldos.saldo+dados.limite) >= sqlc.arg(valor) then true
	  else false
	end as autorizado,	  
	dados.id_conta,saldos.saldo,dados.limite,(saldos.saldo+dados.limite) as disponivel from dados
	join saldos on dados.id_conta=saldos.id_conta
	for update
),
credito as (
	update saldos ss set saldo=ss.saldo+sqlc.arg(valor)
	from informacoes i
	where ss.id_conta=i.id_conta
	returning ss.*,i.limite
),
inserehistorico as (
    insert into transacoes (id_conta,tipo_operacao,valor,descricao)
	select id_conta, 'c', sqlc.arg(valor), sqlc.arg(descricao)::text from informacoes where informacoes.autorizado
	returning *
)
select inf.autorizado, inf.id_conta, inf.saldo as saldo_anterior, inf.limite, coalesce(c.saldo,inf.saldo) as saldo, sqlc.arg(valor)::bigint as valor, sqlc.arg(descricao)::text as descricao
from informacoes inf
left join credito c on inf.id_conta=c.id_conta
left join inserehistorico i on c.id_conta=i.id_conta;

-- name: Debito :one
with dados as (
	select * from dados_bancarios 
	where dados_bancarios.id_conta=sqlc.arg(id_conta)
	for update
),
informacoes as (
    select 
	case
	  when (saldos.saldo+dados.limite) >= sqlc.arg(valor) then true
	  else false
	end as autorizado,	  
	dados.id_conta,saldos.saldo,dados.limite,(saldos.saldo+dados.limite) as disponivel from dados
	join saldos on dados.id_conta=saldos.id_conta
	for update
),
debito as (
	update saldos ss set saldo=ss.saldo-sqlc.arg(valor)
	from informacoes i
	where ss.id_conta=i.id_conta and i.autorizado
	returning ss.*,i.limite
),
inserehistorico as (
    insert into transacoes (id_conta,tipo_operacao,valor,descricao)
	select id_conta, 'd', sqlc.arg(valor), sqlc.arg(descricao)::text from informacoes where informacoes.autorizado
	returning *
)
select inf.autorizado, inf.id_conta, inf.saldo as saldo_anterior, inf.limite, coalesce(d.saldo,inf.saldo) as saldo, sqlc.arg(valor)::bigint as valor, sqlc.arg(descricao)::text as descricao
from informacoes inf
left join debito d on inf.id_conta=d.id_conta
left join inserehistorico i on d.id_conta=i.id_conta;

-- name: Extrato :many
with dados as (
	select * from dados_bancarios 
	where dados_bancarios.id_conta=sqlc.arg(id_conta)
	for update
),
informacoes as (
    select dados.id_conta,saldos.saldo,dados.limite,(saldos.saldo+dados.limite) as disponivel from dados
	join saldos on dados.id_conta=saldos.id_conta
),
extrato as (
	select * from transacoes
	order by id desc
)
select i.saldo, now()::timestamp without time zone as data_extrato, i.limite, COALESCE(e.tipo_operacao, 'e')::text as tipo_operacao, e.valor, COALESCE(e.descricao,'')::text as descricao, e.created_at as realizada_em
from informacoes i left join extrato e on e.id_conta=i.id_conta
order by e.id desc limit 10;