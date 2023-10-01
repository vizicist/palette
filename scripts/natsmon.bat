if "%1" == "" "%PALETTE%\bin\nats-sub" -s "%NATS_USER%:%NATS_PASSWORD%@%NATS_URL%" ">"
if not "%1" == "" "%PALETTE%\bin\nats-sub" -s "%NATS_USER%:%NATS_PASSWORD%@%NATS_URL%" %1
