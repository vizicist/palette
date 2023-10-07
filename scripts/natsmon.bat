if "%1" == "" "%PALETTE%\bin\nats" -s "%NATS_USER%:%NATS_PASSWORD%@%NATS_URL%" subscribe ">"
if not "%1" == "" "%PALETTE%\bin\nats" -s "%NATS_USER%:%NATS_PASSWORD%@%NATS_URL%" subscribe %1
