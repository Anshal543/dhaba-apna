-- Dhaba Orders schema and seed data.
-- Applied once, when the Postgres volume is first created.
-- menu-api reads menu_items. orders-api stores each order as one row
-- (items is a JSON array). kitchen-worker advances status and updated_at.

CREATE TABLE menu_items (
    id          SERIAL PRIMARY KEY,
    name        TEXT        NOT NULL UNIQUE,
    description TEXT        NOT NULL DEFAULT '',
    category    TEXT        NOT NULL,
    price_pkr   INTEGER     NOT NULL CHECK (price_pkr > 0),
    available   BOOLEAN     NOT NULL DEFAULT TRUE
);

CREATE TABLE orders (
    id          BIGSERIAL PRIMARY KEY,
    customer    TEXT        NOT NULL,
    items       JSONB       NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'new'
                CHECK (status IN ('new', 'cooking', 'ready')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_status ON orders (status);

INSERT INTO menu_items (name, description, category, price_pkr) VALUES
 ('Chicken Karahi',  'Wok-cooked chicken with tomato, ginger and green chilli', 'Mains',   1800),
 ('Mutton Karahi',   'Slow-cooked mutton karahi, Peshawari style',               'Mains',   2800),
 ('Daal Tadka',      'Yellow lentils with a garlic and cumin tarka',             'Mains',    450),
 ('Chapli Kebab',    'Spiced minced beef patty, pan fried',                      'Starters', 700),
 ('Seekh Kebab',     'Charcoal-grilled minced beef skewers (4 pcs)',             'Starters', 800),
 ('Chicken Tikka',   'Marinated chicken leg, grilled',                           'Starters', 650),
 ('Butter Naan',     'Tandoor-baked bread brushed with butter',                  'Bread',     80),
 ('Roghni Naan',     'Naan topped with sesame seeds',                            'Bread',    100),
 ('Sweet Lassi',     'Chilled yoghurt drink',                                    'Drinks',   250),
 ('Doodh Patti Chai','Strong milky tea',                                         'Drinks',   120),
 ('Gulab Jamun',     'Milk dumplings in sugar syrup (2 pcs)',                    'Dessert',  200);
