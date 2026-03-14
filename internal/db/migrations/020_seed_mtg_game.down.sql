DELETE FROM card_games WHERE name = 'Magic: The Gathering' AND id NOT IN (SELECT card_game_id FROM cards);
