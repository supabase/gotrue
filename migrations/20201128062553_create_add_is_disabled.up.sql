ALTER TABLE `{{ index .Options "Namespace" }}users` ADD `is_disabled` tinyint(1) NULL DEFAULT NULL AFTER `invited_at`;
