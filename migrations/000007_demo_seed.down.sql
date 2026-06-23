-- =============================================================================
-- ROLLBACK DU SEED DE DÉMO
-- -----------------------------------------------------------------------------
-- On supprime les comptes de démo (domaine @duskforge.demo) ; les cascades
-- ON DELETE CASCADE du schéma se chargent de tout le reste (reviews, likes,
-- commentaires, follows, messages, notifications, activités, succès, etc.).
-- Le super-admin (hors @duskforge.demo) et le catalogue de succès (table
-- achievements, migration 000005) sont PRÉSERVÉS.
--
-- Problème de perf : plusieurs colonnes de clé étrangère (notamment sur les
-- grosses tables notifications/activities) ne sont pas indexées. Les triggers
-- d'intégrité référentielle s'exécutant ligne par ligne, chaque suppression en
-- cascade ferait un parcours séquentiel complet → des heures sur des centaines
-- de milliers de lignes. On crée donc des index TEMPORAIRES sur ces colonnes le
-- temps de la suppression, puis on les retire.
-- =============================================================================

DO $undo$
BEGIN
    -- index temporaires de support des cascades
    CREATE INDEX IF NOT EXISTS tmp_dn_notif_review     ON notifications(review_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_notif_comment    ON notifications(comment_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_notif_user       ON notifications(user_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_notif_actor      ON notifications(actor_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_notif_ach        ON notifications(achievement_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_act_review       ON activities(review_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_act_comment      ON activities(comment_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_act_collection   ON activities(collection_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_act_user         ON activities(user_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_act_target       ON activities(target_user_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_comments_review  ON comments(review_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_comments_user    ON comments(user_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_rlikes_review    ON review_likes(review_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_clikes_comment   ON comment_likes(comment_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_follows_follow   ON follows(following_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_msg_sender       ON messages(sender_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_msg_receiver     ON messages(receiver_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_sessions_user    ON sessions(user_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_mreact_user      ON message_reactions(user_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_convstate_other  ON conversation_states(other_user_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_citems_coll      ON collection_items(collection_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_reports_review   ON reports(target_review_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_reports_comment  ON reports(target_comment_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_reports_tuser    ON reports(target_user_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_reports_reporter ON reports(reporter_id);
    CREATE INDEX IF NOT EXISTS tmp_dn_reports_resolver ON reports(resolver_id);

    -- suppression ciblée — les cascades font le reste
    DELETE FROM users WHERE email LIKE '%@duskforge.demo';

    -- retrait des index temporaires
    DROP INDEX IF EXISTS
        tmp_dn_notif_review, tmp_dn_notif_comment, tmp_dn_notif_user, tmp_dn_notif_actor, tmp_dn_notif_ach,
        tmp_dn_act_review, tmp_dn_act_comment, tmp_dn_act_collection, tmp_dn_act_user, tmp_dn_act_target,
        tmp_dn_comments_review, tmp_dn_comments_user, tmp_dn_rlikes_review, tmp_dn_clikes_comment,
        tmp_dn_follows_follow, tmp_dn_msg_sender, tmp_dn_msg_receiver, tmp_dn_sessions_user,
        tmp_dn_mreact_user, tmp_dn_convstate_other, tmp_dn_citems_coll,
        tmp_dn_reports_review, tmp_dn_reports_comment, tmp_dn_reports_tuser, tmp_dn_reports_reporter, tmp_dn_reports_resolver;
END
$undo$;
