-- The board version at which a card last changed place: moved, or pushed
-- along its lane by a move into it or out of it, or put back by the reset.
-- The board's frames give a view-transition name only to the cards that
-- changed place lately, which glide; the others hold still at no cost.
ALTER TABLE board_cards ADD COLUMN shifted INTEGER NOT NULL DEFAULT 0;
