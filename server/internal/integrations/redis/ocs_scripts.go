package redis

import redisv9 "github.com/redis/go-redis/v9"

const ocsLuaHelpers = `
local function result(
  code,
  stream_id,
  wallet_version,
  charge_sequence,
  balance,
  wallet_reserved,
  authorized,
  consumed,
  charge_reserved,
  charge_status
)
  return {
    tostring(code or ''),
    tostring(stream_id or ''),
    tostring(wallet_version or 0),
    tostring(charge_sequence or 0),
    tostring(balance or 0),
    tostring(wallet_reserved or 0),
    tostring(authorized or 0),
    tostring(consumed or 0),
    tostring(charge_reserved or 0),
    tostring(charge_status or '')
  }
end

local function operation_replay(
  key,
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status
)
  if redis.call('EXISTS', key) == 0 then
    return nil
  end

  if redis.call('HGET', key, 'operation') ~= operation
    or redis.call('HGET', key, 'wallet_id') ~= wallet_id
    or redis.call('HGET', key, 'charge_id') ~= charge_id
    or redis.call('HGET', key, 'amount_micros') ~= tostring(amount)
    or redis.call('HGET', key, 'terminal_status') ~= terminal_status then
    return result('operation_conflict', '', 0, 0, 0, 0, 0, 0, 0, '')
  end

  return result(
    'replay',
    redis.call('HGET', key, 'stream_id'),
    redis.call('HGET', key, 'wallet_version'),
    redis.call('HGET', key, 'charge_sequence'),
    redis.call('HGET', key, 'balance_micros'),
    redis.call('HGET', key, 'wallet_reserved_micros'),
    redis.call('HGET', key, 'charge_authorized_micros'),
    redis.call('HGET', key, 'charge_consumed_micros'),
    redis.call('HGET', key, 'charge_reserved_micros'),
    redis.call('HGET', key, 'charge_status')
  )
end

local function store_operation(
  key,
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status,
  stream_id,
  wallet_version,
  charge_sequence,
  balance,
  wallet_reserved,
  authorized,
  consumed,
  charge_reserved,
  charge_status
)
  redis.call(
    'HSET',
    key,
    'operation', operation,
    'wallet_id', wallet_id,
    'charge_id', charge_id,
    'amount_micros', tostring(amount),
    'terminal_status', terminal_status,
    'stream_id', stream_id,
    'wallet_version', tostring(wallet_version),
    'charge_sequence', tostring(charge_sequence),
    'balance_micros', tostring(balance),
    'wallet_reserved_micros', tostring(wallet_reserved),
    'charge_authorized_micros', tostring(authorized),
    'charge_consumed_micros', tostring(consumed),
    'charge_reserved_micros', tostring(charge_reserved),
    'charge_status', charge_status
  )
end

local function wallet_state(wallet_key, organization_id, wallet_id, currency)
  if redis.call('EXISTS', wallet_key) == 0 then
    return nil, result('wallet_not_found', '', 0, 0, 0, 0, 0, 0, 0, '')
  end

  if redis.call('HGET', wallet_key, 'organization_id') ~= organization_id
    or redis.call('HGET', wallet_key, 'wallet_id') ~= wallet_id
    or redis.call('HGET', wallet_key, 'currency') ~= currency then
    return nil, result('wallet_mismatch', '', 0, 0, 0, 0, 0, 0, 0, '')
  end

  return {
    status = redis.call('HGET', wallet_key, 'status'),
    balance = tonumber(redis.call('HGET', wallet_key, 'balance_micros')) or 0,
    reserved = tonumber(redis.call('HGET', wallet_key, 'reserved_micros')) or 0,
    version = tonumber(redis.call('HGET', wallet_key, 'version')) or 0
  }, nil
end

local function charge_state(charge_key)
  if redis.call('EXISTS', charge_key) == 0 then
    return nil
  end

  return {
    organization_id = redis.call('HGET', charge_key, 'organization_id'),
    wallet_id = redis.call('HGET', charge_key, 'wallet_id'),
    currency = redis.call('HGET', charge_key, 'currency'),
    charging_mode = redis.call('HGET', charge_key, 'charging_mode'),
    status = redis.call('HGET', charge_key, 'status'),
    authorized = tonumber(redis.call('HGET', charge_key, 'authorized_micros')) or 0,
    consumed = tonumber(redis.call('HGET', charge_key, 'consumed_micros')) or 0,
    reserved = tonumber(redis.call('HGET', charge_key, 'reserved_micros')) or 0,
    sequence = tonumber(redis.call('HGET', charge_key, 'sequence')) or 0
  }
end

local function charge_matches(
  charge,
  organization_id,
  wallet_id,
  currency
)
  return charge.organization_id == organization_id
    and charge.wallet_id == wallet_id
    and charge.currency == currency
end

local function append_event(
  stream_key,
  organization_id,
  wallet_id,
  charge_id,
  operation_id,
  wallet_version,
  charge_sequence,
  event_type,
  balance_delta,
  reserved_delta,
  balance_after,
  wallet_reserved_after,
  charge_authorized_after,
  charge_consumed_after,
  charge_reserved_after,
  charge_status,
  occurred_at
)
  return redis.call(
    'XADD',
    stream_key,
    '*',
    'organization_id', organization_id,
    'wallet_id', wallet_id,
    'charge_id', charge_id,
    'operation_id', operation_id,
    'wallet_version', tostring(wallet_version),
    'charge_sequence', tostring(charge_sequence),
    'event_type', event_type,
    'balance_delta_micros', tostring(balance_delta),
    'reserved_delta_micros', tostring(reserved_delta),
    'balance_after_micros', tostring(balance_after),
    'reserved_after_micros', tostring(wallet_reserved_after),
    'charge_authorized_after_micros', tostring(charge_authorized_after),
    'charge_consumed_after_micros', tostring(charge_consumed_after),
    'charge_reserved_after_micros', tostring(charge_reserved_after),
    'charge_status', charge_status,
    'occurred_at_millis', tostring(occurred_at)
  )
end
`

var ensureOCSWalletScript = redisv9.NewScript(ocsLuaHelpers + `
local organization_id = ARGV[1]
local wallet_id = ARGV[2]
local currency = ARGV[3]
local status = ARGV[4]
local balance = tonumber(ARGV[5])
local reserved = tonumber(ARGV[6])
local version = tonumber(ARGV[7])

if redis.call('EXISTS', KEYS[1]) == 1 then
  if redis.call('HGET', KEYS[1], 'organization_id') ~= organization_id
    or redis.call('HGET', KEYS[1], 'wallet_id') ~= wallet_id
    or redis.call('HGET', KEYS[1], 'currency') ~= currency then
    return result('wallet_mismatch', '', 0, 0, 0, 0, 0, 0, 0, '')
  end

  return result(
    'exists',
    '',
    redis.call('HGET', KEYS[1], 'version'),
    0,
    redis.call('HGET', KEYS[1], 'balance_micros'),
    redis.call('HGET', KEYS[1], 'reserved_micros'),
    0,
    0,
    0,
    ''
  )
end

redis.call(
  'HSET',
  KEYS[1],
  'organization_id', organization_id,
  'wallet_id', wallet_id,
  'currency', currency,
  'status', status,
  'balance_micros', tostring(balance),
  'reserved_micros', tostring(reserved),
  'version', tostring(version)
)

return result(
  'ok',
  '',
  version,
  0,
  balance,
  reserved,
  0,
  0,
  0,
  ''
)
`)

var reserveOCSScript = redisv9.NewScript(ocsLuaHelpers + `
local organization_id = ARGV[1]
local wallet_id = ARGV[2]
local currency = ARGV[3]
local charge_id = ARGV[4]
local operation_id = ARGV[5]
local amount = tonumber(ARGV[6])
local charging_mode = ARGV[7]
local terminal_status = ARGV[8]
local occurred_at = ARGV[9]
local operation = ARGV[10]
local max_micros = tonumber(ARGV[11])

local replay = operation_replay(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status
)
if replay then
  return replay
end

local wallet, wallet_error = wallet_state(
  KEYS[1],
  organization_id,
  wallet_id,
  currency
)
if wallet_error then
  return wallet_error
end
if wallet.status ~= 'active' then
  return result(
    'wallet_not_active',
    '',
    wallet.version,
    0,
    wallet.balance,
    wallet.reserved,
    0,
    0,
    0,
    ''
  )
end

local charge = charge_state(KEYS[2])
if not charge then
  charge = {
    organization_id = organization_id,
    wallet_id = wallet_id,
    currency = currency,
    charging_mode = charging_mode,
    status = 'active',
    authorized = 0,
    consumed = 0,
    reserved = 0,
    sequence = 0
  }
else
  if not charge_matches(
    charge,
    organization_id,
    wallet_id,
    currency
  ) then
    return result(
      'charge_mismatch',
      '',
      wallet.version,
      charge.sequence,
      wallet.balance,
      wallet.reserved,
      charge.authorized,
      charge.consumed,
      charge.reserved,
      charge.status
    )
  end
  if charge.status ~= 'active' then
    return result(
      'charge_not_active',
      '',
      wallet.version,
      charge.sequence,
      wallet.balance,
      wallet.reserved,
      charge.authorized,
      charge.consumed,
      charge.reserved,
      charge.status
    )
  end
  if charge.charging_mode ~= charging_mode then
    return result(
      'charge_mode_mismatch',
      '',
      wallet.version,
      charge.sequence,
      wallet.balance,
      wallet.reserved,
      charge.authorized,
      charge.consumed,
      charge.reserved,
      charge.status
    )
  end
end

local available = wallet.balance - wallet.reserved
if amount > available then
  return result(
    'insufficient_funds',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end
if charge.authorized > max_micros - amount
  or wallet.reserved > max_micros - amount
  or charge.reserved > max_micros - amount then
  return result(
    'overflow',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end

local wallet_version = wallet.version + 1
local charge_sequence = charge.sequence + 1
local wallet_reserved = wallet.reserved + amount
local authorized = charge.authorized + amount
local charge_reserved = charge.reserved + amount

redis.call(
  'HSET',
  KEYS[1],
  'reserved_micros', tostring(wallet_reserved),
  'version', tostring(wallet_version)
)
redis.call(
  'HSET',
  KEYS[2],
  'organization_id', organization_id,
  'wallet_id', wallet_id,
  'currency', currency,
  'charging_mode', charging_mode,
  'status', 'active',
  'authorized_micros', tostring(authorized),
  'consumed_micros', tostring(charge.consumed),
  'reserved_micros', tostring(charge_reserved),
  'sequence', tostring(charge_sequence)
)

local stream_id = append_event(
  KEYS[4],
  organization_id,
  wallet_id,
  charge_id,
  operation_id,
  wallet_version,
  charge_sequence,
  'reserve',
  0,
  amount,
  wallet.balance,
  wallet_reserved,
  authorized,
  charge.consumed,
  charge_reserved,
  'active',
  occurred_at
)

store_operation(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status,
  stream_id,
  wallet_version,
  charge_sequence,
  wallet.balance,
  wallet_reserved,
  authorized,
  charge.consumed,
  charge_reserved,
  'active'
)

return result(
  'ok',
  stream_id,
  wallet_version,
  charge_sequence,
  wallet.balance,
  wallet_reserved,
  authorized,
  charge.consumed,
  charge_reserved,
  'active'
)
`)

var consumeOCSScript = redisv9.NewScript(ocsLuaHelpers + `
local organization_id = ARGV[1]
local wallet_id = ARGV[2]
local currency = ARGV[3]
local charge_id = ARGV[4]
local operation_id = ARGV[5]
local amount = tonumber(ARGV[6])
local terminal_status = ARGV[8]
local occurred_at = ARGV[9]
local operation = ARGV[10]

local replay = operation_replay(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status
)
if replay then
  return replay
end

local wallet, wallet_error = wallet_state(
  KEYS[1],
  organization_id,
  wallet_id,
  currency
)
if wallet_error then
  return wallet_error
end

local charge = charge_state(KEYS[2])
if not charge then
  return result(
    'charge_not_found',
    '',
    wallet.version,
    0,
    wallet.balance,
    wallet.reserved,
    0,
    0,
    0,
    ''
  )
end
if not charge_matches(
  charge,
  organization_id,
  wallet_id,
  currency
) then
  return result(
    'charge_mismatch',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end
if charge.status ~= 'active' then
  return result(
    'charge_not_active',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end
if amount > charge.reserved
  or amount > wallet.reserved
  or amount > wallet.balance then
  return result(
    'insufficient_reservation',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end

local wallet_version = wallet.version + 1
local charge_sequence = charge.sequence + 1
local balance = wallet.balance - amount
local wallet_reserved = wallet.reserved - amount
local consumed = charge.consumed + amount
local charge_reserved = charge.reserved - amount

redis.call(
  'HSET',
  KEYS[1],
  'balance_micros', tostring(balance),
  'reserved_micros', tostring(wallet_reserved),
  'version', tostring(wallet_version)
)
redis.call(
  'HSET',
  KEYS[2],
  'consumed_micros', tostring(consumed),
  'reserved_micros', tostring(charge_reserved),
  'sequence', tostring(charge_sequence)
)

local stream_id = append_event(
  KEYS[4],
  organization_id,
  wallet_id,
  charge_id,
  operation_id,
  wallet_version,
  charge_sequence,
  'consume',
  -amount,
  -amount,
  balance,
  wallet_reserved,
  charge.authorized,
  consumed,
  charge_reserved,
  charge.status,
  occurred_at
)

store_operation(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status,
  stream_id,
  wallet_version,
  charge_sequence,
  balance,
  wallet_reserved,
  charge.authorized,
  consumed,
  charge_reserved,
  charge.status
)

return result(
  'ok',
  stream_id,
  wallet_version,
  charge_sequence,
  balance,
  wallet_reserved,
  charge.authorized,
  consumed,
  charge_reserved,
  charge.status
)
`)

var releaseOCSScript = redisv9.NewScript(ocsLuaHelpers + `
local organization_id = ARGV[1]
local wallet_id = ARGV[2]
local currency = ARGV[3]
local charge_id = ARGV[4]
local operation_id = ARGV[5]
local amount = tonumber(ARGV[6])
local terminal_status = ARGV[8]
local occurred_at = ARGV[9]
local operation = ARGV[10]

local replay = operation_replay(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status
)
if replay then
  return replay
end

local wallet, wallet_error = wallet_state(
  KEYS[1],
  organization_id,
  wallet_id,
  currency
)
if wallet_error then
  return wallet_error
end

local charge = charge_state(KEYS[2])
if not charge then
  return result(
    'charge_not_found',
    '',
    wallet.version,
    0,
    wallet.balance,
    wallet.reserved,
    0,
    0,
    0,
    ''
  )
end
if not charge_matches(
  charge,
  organization_id,
  wallet_id,
  currency
) then
  return result(
    'charge_mismatch',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end
if charge.status ~= 'active' then
  return result(
    'charge_not_active',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end
if amount > charge.reserved
  or amount > wallet.reserved then
  return result(
    'insufficient_reservation',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end

local wallet_version = wallet.version + 1
local charge_sequence = charge.sequence + 1
local wallet_reserved = wallet.reserved - amount
local charge_reserved = charge.reserved - amount

redis.call(
  'HSET',
  KEYS[1],
  'reserved_micros', tostring(wallet_reserved),
  'version', tostring(wallet_version)
)
redis.call(
  'HSET',
  KEYS[2],
  'reserved_micros', tostring(charge_reserved),
  'sequence', tostring(charge_sequence)
)

local stream_id = append_event(
  KEYS[4],
  organization_id,
  wallet_id,
  charge_id,
  operation_id,
  wallet_version,
  charge_sequence,
  'release',
  0,
  -amount,
  wallet.balance,
  wallet_reserved,
  charge.authorized,
  charge.consumed,
  charge_reserved,
  charge.status,
  occurred_at
)

store_operation(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status,
  stream_id,
  wallet_version,
  charge_sequence,
  wallet.balance,
  wallet_reserved,
  charge.authorized,
  charge.consumed,
  charge_reserved,
  charge.status
)

return result(
  'ok',
  stream_id,
  wallet_version,
  charge_sequence,
  wallet.balance,
  wallet_reserved,
  charge.authorized,
  charge.consumed,
  charge_reserved,
  charge.status
)
`)

var debitOCSScript = redisv9.NewScript(ocsLuaHelpers + `
local organization_id = ARGV[1]
local wallet_id = ARGV[2]
local currency = ARGV[3]
local charge_id = ARGV[4]
local operation_id = ARGV[5]
local amount = tonumber(ARGV[6])
local charging_mode = ARGV[7]
local terminal_status = ARGV[8]
local occurred_at = ARGV[9]
local operation = ARGV[10]
local max_micros = tonumber(ARGV[11])

local replay = operation_replay(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status
)
if replay then
  return replay
end

local wallet, wallet_error = wallet_state(
  KEYS[1],
  organization_id,
  wallet_id,
  currency
)
if wallet_error then
  return wallet_error
end
if wallet.status ~= 'active' then
  return result(
    'wallet_not_active',
    '',
    wallet.version,
    0,
    wallet.balance,
    wallet.reserved,
    0,
    0,
    0,
    ''
  )
end

local charge = charge_state(KEYS[2])
if not charge then
  charge = {
    organization_id = organization_id,
    wallet_id = wallet_id,
    currency = currency,
    charging_mode = charging_mode,
    status = 'active',
    authorized = 0,
    consumed = 0,
    reserved = 0,
    sequence = 0
  }
else
  if not charge_matches(
    charge,
    organization_id,
    wallet_id,
    currency
  ) then
    return result(
      'charge_mismatch',
      '',
      wallet.version,
      charge.sequence,
      wallet.balance,
      wallet.reserved,
      charge.authorized,
      charge.consumed,
      charge.reserved,
      charge.status
    )
  end
  if charge.status ~= 'active' then
    return result(
      'charge_not_active',
      '',
      wallet.version,
      charge.sequence,
      wallet.balance,
      wallet.reserved,
      charge.authorized,
      charge.consumed,
      charge.reserved,
      charge.status
    )
  end
  if charge.charging_mode ~= 'discrete' then
    return result(
      'charge_mode_mismatch',
      '',
      wallet.version,
      charge.sequence,
      wallet.balance,
      wallet.reserved,
      charge.authorized,
      charge.consumed,
      charge.reserved,
      charge.status
    )
  end
end

local available = wallet.balance - wallet.reserved
if amount > available then
  return result(
    'insufficient_funds',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end
if charge.authorized > max_micros - amount
  or charge.consumed > max_micros - amount then
  return result(
    'overflow',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end

local wallet_version = wallet.version + 1
local charge_sequence = charge.sequence + 1
local balance = wallet.balance - amount
local authorized = charge.authorized + amount
local consumed = charge.consumed + amount

redis.call(
  'HSET',
  KEYS[1],
  'balance_micros', tostring(balance),
  'version', tostring(wallet_version)
)
redis.call(
  'HSET',
  KEYS[2],
  'organization_id', organization_id,
  'wallet_id', wallet_id,
  'currency', currency,
  'charging_mode', 'discrete',
  'status', 'active',
  'authorized_micros', tostring(authorized),
  'consumed_micros', tostring(consumed),
  'reserved_micros', tostring(charge.reserved),
  'sequence', tostring(charge_sequence)
)

local stream_id = append_event(
  KEYS[4],
  organization_id,
  wallet_id,
  charge_id,
  operation_id,
  wallet_version,
  charge_sequence,
  'debit',
  -amount,
  0,
  balance,
  wallet.reserved,
  authorized,
  consumed,
  charge.reserved,
  'active',
  occurred_at
)

store_operation(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status,
  stream_id,
  wallet_version,
  charge_sequence,
  balance,
  wallet.reserved,
  authorized,
  consumed,
  charge.reserved,
  'active'
)

return result(
  'ok',
  stream_id,
  wallet_version,
  charge_sequence,
  balance,
  wallet.reserved,
  authorized,
  consumed,
  charge.reserved,
  'active'
)
`)

var finalizeOCSScript = redisv9.NewScript(ocsLuaHelpers + `
local organization_id = ARGV[1]
local wallet_id = ARGV[2]
local currency = ARGV[3]
local charge_id = ARGV[4]
local operation_id = ARGV[5]
local amount = tonumber(ARGV[6])
local terminal_status = ARGV[8]
local occurred_at = ARGV[9]
local operation = ARGV[10]

local replay = operation_replay(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status
)
if replay then
  return replay
end

local wallet, wallet_error = wallet_state(
  KEYS[1],
  organization_id,
  wallet_id,
  currency
)
if wallet_error then
  return wallet_error
end

local charge = charge_state(KEYS[2])
if not charge then
  return result(
    'charge_not_found',
    '',
    wallet.version,
    0,
    wallet.balance,
    wallet.reserved,
    0,
    0,
    0,
    ''
  )
end
if not charge_matches(
  charge,
  organization_id,
  wallet_id,
  currency
) then
  return result(
    'charge_mismatch',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end
if charge.status ~= 'active' then
  return result(
    'charge_not_active',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end
if charge.reserved > wallet.reserved then
  return result(
    'wallet_mismatch',
    '',
    wallet.version,
    charge.sequence,
    wallet.balance,
    wallet.reserved,
    charge.authorized,
    charge.consumed,
    charge.reserved,
    charge.status
  )
end

local wallet_version = wallet.version + 1
local charge_sequence = charge.sequence + 1
local wallet_reserved = wallet.reserved - charge.reserved
local released = charge.reserved

redis.call(
  'HSET',
  KEYS[1],
  'reserved_micros', tostring(wallet_reserved),
  'version', tostring(wallet_version)
)
redis.call(
  'HSET',
  KEYS[2],
  'status', terminal_status,
  'reserved_micros', '0',
  'sequence', tostring(charge_sequence)
)

local stream_id = append_event(
  KEYS[4],
  organization_id,
  wallet_id,
  charge_id,
  operation_id,
  wallet_version,
  charge_sequence,
  'finalize',
  0,
  -released,
  wallet.balance,
  wallet_reserved,
  charge.authorized,
  charge.consumed,
  0,
  terminal_status,
  occurred_at
)

store_operation(
  KEYS[3],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status,
  stream_id,
  wallet_version,
  charge_sequence,
  wallet.balance,
  wallet_reserved,
  charge.authorized,
  charge.consumed,
  0,
  terminal_status
)

return result(
  'ok',
  stream_id,
  wallet_version,
  charge_sequence,
  wallet.balance,
  wallet_reserved,
  charge.authorized,
  charge.consumed,
  0,
  terminal_status
)
`)

var creditOCSScript = redisv9.NewScript(ocsLuaHelpers + `
local organization_id = ARGV[1]
local wallet_id = ARGV[2]
local currency = ARGV[3]
local operation_id = ARGV[4]
local amount = tonumber(ARGV[5])
local occurred_at = ARGV[6]
local max_micros = tonumber(ARGV[7])
local operation = 'credit'
local charge_id = ''
local terminal_status = ''

local replay = operation_replay(
  KEYS[2],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status
)
if replay then
  return replay
end

local wallet, wallet_error = wallet_state(
  KEYS[1],
  organization_id,
  wallet_id,
  currency
)
if wallet_error then
  return wallet_error
end
if wallet.status == 'closed' then
  return result(
    'wallet_not_active',
    '',
    wallet.version,
    0,
    wallet.balance,
    wallet.reserved,
    0,
    0,
    0,
    ''
  )
end
if wallet.balance > max_micros - amount then
  return result(
    'overflow',
    '',
    wallet.version,
    0,
    wallet.balance,
    wallet.reserved,
    0,
    0,
    0,
    ''
  )
end

local wallet_version = wallet.version + 1
local balance = wallet.balance + amount

redis.call(
  'HSET',
  KEYS[1],
  'balance_micros', tostring(balance),
  'version', tostring(wallet_version)
)

local stream_id = append_event(
  KEYS[3],
  organization_id,
  wallet_id,
  '',
  operation_id,
  wallet_version,
  0,
  'credit',
  amount,
  0,
  balance,
  wallet.reserved,
  0,
  0,
  0,
  '',
  occurred_at
)

store_operation(
  KEYS[2],
  operation,
  wallet_id,
  charge_id,
  amount,
  terminal_status,
  stream_id,
  wallet_version,
  0,
  balance,
  wallet.reserved,
  0,
  0,
  0,
  ''
)

return result(
  'ok',
  stream_id,
  wallet_version,
  0,
  balance,
  wallet.reserved,
  0,
  0,
  0,
  ''
)
`)
