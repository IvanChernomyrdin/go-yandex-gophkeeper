-- удаляем типы секретов и таблицу секретов с принадлежащими ей индексами
DROP TABLE IF EXISTS secrets;
DROP TYPE IF EXISTS secret_type;