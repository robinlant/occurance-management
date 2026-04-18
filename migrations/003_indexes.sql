CREATE INDEX IF NOT EXISTS idx_occurrences_date ON occurrences(date);
CREATE INDEX IF NOT EXISTS idx_occurrences_group_id ON occurrences(group_id);
CREATE INDEX IF NOT EXISTS idx_participations_occurrence_id ON participations(occurrence_id);
CREATE INDEX IF NOT EXISTS idx_out_of_office_user_id ON out_of_office(user_id);
