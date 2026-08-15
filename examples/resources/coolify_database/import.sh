# The API's database object does not carry its engine directly (several
# engines are ambiguous from the image name alone — redis, keydb and
# dragonfly all speak the Redis protocol), so the engine rides in the import
# ID: <engine>/<uuid>. Valid engines: postgresql, mysql, mariadb, mongodb,
# redis, keydb, dragonfly, clickhouse.
terraform import coolify_database.example postgresql/<uuid>
