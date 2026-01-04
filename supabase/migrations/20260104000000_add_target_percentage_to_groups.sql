-- Add target_percentage to groups table
ALTER TABLE groups 
ADD COLUMN IF NOT EXISTS target_percentage NUMERIC DEFAULT 0.0;
