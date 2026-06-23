-- =============================================================================
-- DEMO SEED — données volumineuses et réalistes pour la soutenance DuskForge
-- -----------------------------------------------------------------------------
-- Cette migration peuple la base avec ~5000 utilisateurs et tout l'écosystème
-- social associé (reviews, likes, commentaires, follows, collections, films
-- vus, messages, notifications, activités, succès, signalements).
--
-- IDENTIFICATION : tous les comptes de démo utilisent le domaine e-mail
--   @duskforge.demo  → la migration .down.sql les supprime proprement
--   (les suppressions en cascade nettoient tout le reste).
--
-- GARDE-FOU : le seed ne s'exécute QUE si aucun compte @duskforge.demo n'existe
--   déjà. Sur une base de prod contenant de vrais comptes, relancer cette
--   migration ne fera rien d'autre qu'ajouter les comptes de démo une seule
--   fois ; pour une base déjà semée, c'est un no-op.
--
-- CONNEXION : tous les comptes de démo partagent le même mot de passe :
--   email   : <username>@duskforge.demo   (ex: marie.lefilm@duskforge.demo)
--   mot de passe : DuskForge2026!
--   (hash bcrypt coût 12, identique pour tous — non sensible, données fictives)
--
-- Les volumes sont réglables via les variables v_* en tête du bloc.
-- =============================================================================

DO $seed$
DECLARE
    -- ---- volumes (ajustables) -------------------------------------------------
    v_user_count       int := 5000;   -- utilisateurs "ambiants"
    v_review_attempts  int := 30000;  -- tentatives d'insertion de reviews (dédupliquées)
    v_max_follows      int := 40;      -- borne haute de follows par utilisateur
    v_max_likes        int := 22;      -- borne haute de likes par review
    v_max_comments     int := 5;       -- borne haute de commentaires par review
    v_max_clikes       int := 7;       -- borne haute de likes par commentaire

    -- ---- mot de passe partagé : "DuskForge2026!" (bcrypt, coût 12) -----------
    v_pwd  text := '$2a$12$zK5nHECao.eL5gFZ90oui.vwtfT2ZSToxXY1pZjK0NOUVCFiIPNh.';

    -- ---- runtime ---------------------------------------------------------------
    uids     uuid[];
    ucreated timestamptz[];   -- dates de création alignées sur uids (même index)
    v_card int;

    -- comptes nommés pour le scénario de démo (section 18)
    v_marie uuid; v_paul uuid; v_lea uuid; v_theo uuid;
    v_mids int[];
    v_mcard int;

    -- ---- jeux de données réalistes --------------------------------------------
    v_first_names text[] := ARRAY[
        'Marie','Lucas','Emma','Thomas','Lea','Hugo','Chloe','Nathan','Camille','Theo',
        'Sarah','Maxime','Manon','Antoine','Julie','Alexandre','Ines','Quentin','Laura','Romain',
        'Clara','Paul','Sophie','Adrien','Jade','Enzo','Louise','Gabriel','Anais','Mathis',
        'Yasmine','Noah','Lina','Raphael','Zoe','Liam','Aya','Sacha','Nour','Ethan'];
    v_last_names text[] := ARRAY[
        'Martin','Bernard','Dubois','Thomas','Robert','Richard','Petit','Durand','Leroy','Moreau',
        'Simon','Laurent','Lefebvre','Michel','Garcia','David','Bertrand','Roux','Vincent','Fournier',
        'Morel','Girard','Andre','Lefevre','Mercier','Dupont','Lambert','Bonnet','Francois','Martinez'];
    v_bios text[] := ARRAY[
        'Cinéphile invétéré 🎬', 'J''écris sur les films qui me marquent.', 'Team horreur 👻',
        'Fan de SF, de Villeneuve et de plans-séquences', 'Je note tout ce que je regarde',
        '100% films d''auteur', 'Marvel un jour, Marvel toujours 🍿', 'Ciné-club du dimanche soir',
        'Réfugié de Letterboxd', 'Critique amateur, spectateur professionnel',
        'La photographie de film, c''est ma religion', 'Du muet au Dolby Atmos',
        'Thrillers & true crime addict', 'Animation japonaise avant tout ✨',
        'Je collectionne les pépites méconnues', 'Festivalier compulsif',
        'Trois films par jour, ça vous regarde', 'Plutôt VOST jusqu''au bout des ongles',
        NULL, NULL];
    v_reviews text[] := ARRAY[
        'Un chef-d''œuvre absolu. La réalisation est hypnotique du début à la fin.',
        'Visuellement somptueux mais le scénario manque un peu de profondeur.',
        'Je ne m''attendais pas à être autant bouleversé. Une vraie claque émotionnelle.',
        'Le casting est parfait, mention spéciale au second rôle qui crève l''écran.',
        'Trop long à mon goût, j''ai décroché dans le dernier acte.',
        'La bande-originale à elle seule mérite le détour.',
        'Une mise en scène virtuose au service d''un propos universel.',
        'Culte. Je le revois chaque année et il ne vieillit pas d''une ride.',
        'Surcoté. Sympa sans plus, je ne comprends pas l''engouement général.',
        'Un film qui hante longtemps après le générique de fin.',
        'Le rythme est impeccable, on ne voit pas passer les deux heures.',
        'Des dialogues ciselés et une photographie à tomber par terre.',
        'Plus qu''un film, une véritable expérience sensorielle.',
        'Le genre de pépite qu''on a envie de faire découvrir à tout le monde.',
        'Quelques longueurs mais la fin rattrape très largement le reste.',
        'Un classique indémodable, à voir au moins une fois dans sa vie.',
        'Ambiance pesante et maîtrisée, ce n''est pas pour les âmes sensibles.',
        'Drôle, touchant, intelligent : le combo gagnant.',
        'Techniquement irréprochable mais étrangement froid.',
        'Coup de cœur de l''année, sans la moindre hésitation. ⭐'];
    v_comments text[] := ARRAY[
        'Totalement d''accord avec toi !', 'Je n''avais pas vu ça sous cet angle, merci.',
        'Tu es un peu sévère je trouve 😅', 'Exactement ce que je ressentais.',
        'À revoir absolument.', 'La scène de fin... 🤯', 'Pas convaincu pour ma part.',
        'Excellente critique, bien écrite !', 'Tu m''as donné envie de le revoir ce soir.',
        'On en reparle dans 10 ans, culte assuré.', 'Le meilleur de sa filmographie selon moi.',
        'Bof bof, je n''ai vraiment pas accroché.', 'Grosse découverte grâce à toi 🙏',
        'Le genre de film qui divise, et c''est tant mieux.'];
    v_messages text[] := ARRAY[
        'Salut ! Tu as vu le dernier Villeneuve ?', 'Faut absolument que tu regardes ça ce week-end',
        'haha j''ai pensé à toi en le voyant', 'On se fait une soirée ciné bientôt ?',
        'Ta critique est au top 👌', 'Tu me conseilles quoi pour ce soir ?',
        'Je viens de finir, quelle claque', 'Spoiler : la fin est totalement dingue',
        'Merci pour la reco, c''était parfait !', 'On est d''accord, complètement surcoté 😂',
        'T''as réussi à choper la projo de jeudi ?', 'Hâte d''avoir ton avis dessus'];
    v_theme_names text[] := ARRAY[
        'Mes films cultes','À voir un soir de pluie','Pépites méconnues','Marathon horreur',
        'SF & dystopies','Le meilleur de Nolan','Comfort movies','Oscars du futur',
        'Animation japonaise','Thrillers haletants','Mon top all-time','Larmes garanties'];
    v_theme_slugs text[] := ARRAY[
        'mes-films-cultes','a-voir-un-soir-de-pluie','pepites-meconnues','marathon-horreur',
        'sf-et-dystopies','le-meilleur-de-nolan','comfort-movies','oscars-du-futur',
        'animation-japonaise','thrillers-haletants','mon-top-all-time','larmes-garanties'];
    v_reasons report_reason[] := ARRAY['spam','harassment','spoiler','inappropriate','other']::report_reason[];

    v_fn_len    int; v_ln_len int; v_bio_len int; v_rev_len int;
    v_com_len   int; v_msg_len int; v_theme_len int;
    v_users_n   int; v_reviews_n bigint; v_likes_n bigint; v_follows_n bigint;
BEGIN
    -- ---- garde-fou : ne semer qu'une base vierge de données de démo ----------
    IF EXISTS (SELECT 1 FROM users WHERE email LIKE '%@duskforge.demo') THEN
        RAISE NOTICE '[demo seed] données de démo déjà présentes — seed ignoré.';
        RETURN;
    END IF;

    v_fn_len := array_length(v_first_names,1);
    v_ln_len := array_length(v_last_names,1);
    v_bio_len := array_length(v_bios,1);
    v_rev_len := array_length(v_reviews,1);
    v_com_len := array_length(v_comments,1);
    v_msg_len := array_length(v_messages,1);
    v_theme_len := array_length(v_theme_names,1);

    RAISE NOTICE '[demo seed] démarrage du peuplement (~% utilisateurs)...', v_user_count;

    -- =========================================================================
    -- 1. CATALOGUE FILMS — vrais tmdb_id + durées réelles (rendu TMDB correct)
    -- =========================================================================
    CREATE TEMP TABLE _seed_movies (tmdb_id int PRIMARY KEY, runtime int) ON COMMIT DROP;
    INSERT INTO _seed_movies (tmdb_id, runtime) VALUES
        (550,139),(603,136),(680,154),(278,142),(155,152),(27205,148),(157336,169),(13,142),
        (238,175),(240,202),(496243,132),(129,125),(120,178),(121,179),(122,201),(98,155),
        (475557,122),(299534,181),(299536,149),(24428,143),(769,145),(807,127),(274,118),
        (244786,106),(120467,99),(313369,128),(68718,165),(16869,153),(106646,180),(424,195),
        (857,169),(597,194),(19995,162),(329,127),(105,116),(348,117),(679,137),(78,117),
        (335984,164),(1422,151),(11324,138),(1124,130),(77,113),(76341,120),(324857,117),
        (354912,105),(14160,96),(10681,98),(862,81),(12,100),(8587,88),(419430,104),(493922,127),
        (546554,130),(530915,119),(374720,106),(466272,162),(6977,122),(7345,158),(64690,100),
        (152601,126),(38,108),(670,120),(396535,118),(372058,106),(128,134),(4935,119),(438631,155),
        (693134,167),(872585,181),(346698,114),(545611,139),(414906,176),(361743,130),(634649,148),
        (245891,101),(210577,149),(146233,153),(37799,120),(389,96),(289,102),(539,109),(103,114),
        (28,147),(62,149);

    SELECT array_agg(tmdb_id ORDER BY tmdb_id) INTO v_mids FROM _seed_movies;
    v_mcard := array_length(v_mids,1);

    -- =========================================================================
    -- 2. COMPTES "HÉROS" — comptes de démo soignés et reconnaissables.
    --    Insérés en premier (created_at le plus ancien) → en tête du tableau
    --    uids → ciblés par le biais de popularité (beaucoup de followers).
    -- =========================================================================
    INSERT INTO users (email, password_hash, username, avatar_url, bio, role, theme, locale, email_verified, created_at, updated_at) VALUES
        ('marie.lefilm@duskforge.demo',      v_pwd, 'marie.lefilm',      'https://i.pravatar.cc/300?img=5',  'Rédactrice ciné 🎬 | je vis pour les plans-séquences', 'admin','dark','fr', TRUE, NOW()-interval '860 days', NOW()-interval '860 days'),
        ('theo.cinephile@duskforge.demo',    v_pwd, 'theo.cinephile',    'https://i.pravatar.cc/300?img=12', 'Team SF & Villeneuve | 1200 films au compteur',        'admin','dark','fr', TRUE, NOW()-interval '858 days', NOW()-interval '858 days'),
        ('lea.popcorn@duskforge.demo',       v_pwd, 'lea.popcorn',       'https://i.pravatar.cc/300?img=20', 'Horreur, giallo et nanars assumés 👻',                  'user', 'dark','fr', TRUE, NOW()-interval '856 days', NOW()-interval '856 days'),
        ('hugo.reels@duskforge.demo',        v_pwd, 'hugo.reels',        'https://i.pravatar.cc/300?img=33', 'Monteur le jour, spectateur la nuit',                  'user', 'system','fr', TRUE, NOW()-interval '854 days', NOW()-interval '854 days'),
        ('chloe.frames@duskforge.demo',      v_pwd, 'chloe.frames',      'https://i.pravatar.cc/300?img=47', 'Animation japonaise > tout ✨',                         'user', 'light','fr', TRUE, NOW()-interval '852 days', NOW()-interval '852 days'),
        ('nathan.reviews@duskforge.demo',    v_pwd, 'nathan.reviews',    'https://i.pravatar.cc/300?img=51', 'Je note tout, je ne pardonne rien',                    'user', 'dark','en', TRUE, NOW()-interval '850 days', NOW()-interval '850 days'),
        ('camille.cinema@duskforge.demo',    v_pwd, 'camille.cinema',    'https://i.pravatar.cc/300?img=9',  'Ciné d''auteur & festivals',                            'user', 'system','fr', TRUE, NOW()-interval '848 days', NOW()-interval '848 days'),
        ('maxime.movies@duskforge.demo',     v_pwd, 'maxime.movies',     'https://i.pravatar.cc/300?img=68', 'Blockbusters totalement assumés 🍿',                    'user', 'dark','fr', TRUE, NOW()-interval '846 days', NOW()-interval '846 days'),
        ('sarah.screen@duskforge.demo',      v_pwd, 'sarah.screen',      'https://i.pravatar.cc/300?img=24', 'Thrillers & true crime',                               'user', 'light','en', TRUE, NOW()-interval '844 days', NOW()-interval '844 days'),
        ('antoine.projection@duskforge.demo',v_pwd, 'antoine.projection','https://i.pravatar.cc/300?img=15', 'Cinéphile du dimanche soir',                           'user', 'system','fr', TRUE, NOW()-interval '842 days', NOW()-interval '842 days'),
        ('julie.cadrage@duskforge.demo',     v_pwd, 'julie.cadrage',     'https://i.pravatar.cc/300?img=44', 'La photo de film = ma religion',                       'user', 'dark','fr', TRUE, NOW()-interval '840 days', NOW()-interval '840 days'),
        ('lucas.bobine@duskforge.demo',      v_pwd, 'lucas.bobine',      'https://i.pravatar.cc/300?img=53', 'Du muet au Dolby Atmos',                               'user', 'dark','es', TRUE, NOW()-interval '838 days', NOW()-interval '838 days'),
        -- compte « compagnon » de démo : utilisateur RÉGULIER (role user), profil
        -- actif, en follow mutuel avec marie.lefilm → peut échanger des messages.
        ('paul.spectateur@duskforge.demo',   v_pwd, 'paul.spectateur',   'https://i.pravatar.cc/300?img=60', 'Spectateur curieux, du blockbuster au film d''auteur 🎟️', 'user', 'dark','fr', TRUE, NOW()-interval '836 days', NOW()-interval '836 days');

    -- =========================================================================
    -- 3. UTILISATEURS AMBIANTS — usernames/emails uniques, profils variés
    -- =========================================================================
    INSERT INTO users (email, password_hash, username, avatar_url, bio, role, theme, locale, email_verified, created_at)
    SELECT
        u.uname || '@duskforge.demo',
        v_pwd,
        u.uname,
        CASE WHEN random() < 0.82 THEN 'https://i.pravatar.cc/300?u=' || u.uname ELSE NULL END,
        v_bios[1 + floor(random()*v_bio_len)::int],
        'user'::user_role,
        (ARRAY['dark','dark','light','system'])[1 + floor(random()*4)::int]::user_theme,
        (ARRAY['fr','fr','fr','en','en','es'])[1 + floor(random()*6)::int]::user_locale,
        random() < 0.94,
        NOW() - interval '15 days' - (random() * interval '700 days')
    FROM generate_series(1, v_user_count) AS g(g)
    CROSS JOIN LATERAL (
        SELECT lower(
            CASE (g % 5)
                WHEN 0 THEN v_first_names[1+floor(random()*v_fn_len)::int]||'.'||v_last_names[1+floor(random()*v_ln_len)::int]
                WHEN 1 THEN v_first_names[1+floor(random()*v_fn_len)::int]||'_'||v_last_names[1+floor(random()*v_ln_len)::int]
                WHEN 2 THEN v_first_names[1+floor(random()*v_fn_len)::int]||v_last_names[1+floor(random()*v_ln_len)::int]
                WHEN 3 THEN v_last_names[1+floor(random()*v_ln_len)::int]||v_first_names[1+floor(random()*v_fn_len)::int]
                ELSE 'the'||v_first_names[1+floor(random()*v_fn_len)::int]
            END
        ) || g::text AS uname
    ) u;

    UPDATE users SET updated_at = created_at WHERE email LIKE '%@duskforge.demo';

    SELECT array_agg(id ORDER BY created_at, id),
           array_agg(created_at ORDER BY created_at, id),
           count(*)
      INTO uids, ucreated, v_card
    FROM users WHERE email LIKE '%@duskforge.demo';
    GET DIAGNOSTICS v_users_n = ROW_COUNT;
    RAISE NOTICE '[demo seed] % utilisateurs créés.', v_card;

    -- =========================================================================
    -- 4. PRÉFÉRENCES DE NOTIFICATION — une ligne par utilisateur (défauts)
    -- =========================================================================
    INSERT INTO notification_preferences (user_id, updated_at)
    SELECT id, created_at FROM users WHERE email LIKE '%@duskforge.demo';

    -- =========================================================================
    -- 5. COLLECTIONS SYSTÈME — "Watched" + "To Watch" pour chaque utilisateur
    --    (normalement créées par l'app à l'inscription ; ici on les recrée)
    -- =========================================================================
    INSERT INTO collections (user_id, name, slug, type, visibility, created_at, updated_at)
    SELECT id, 'Watched', 'watched', 'system', 'public', created_at, created_at
    FROM users WHERE email LIKE '%@duskforge.demo';

    INSERT INTO collections (user_id, name, slug, type, visibility, created_at, updated_at)
    SELECT id, 'To Watch', 'to-watch', 'system', 'public', created_at, created_at
    FROM users WHERE email LIKE '%@duskforge.demo';

    -- collections personnalisées thématiques pour ~12% des utilisateurs
    INSERT INTO collections (user_id, name, slug, type, visibility, description, created_at, updated_at)
    SELECT
        u.id, t.name, t.slug, 'custom',
        CASE WHEN random() < 0.85 THEN 'public' ELSE 'private' END::collection_visibility,
        'Une sélection personnelle, mise à jour au fil de mes visionnages.',
        u.created_at + (random() * (NOW() - u.created_at)),
        NOW()
    FROM users u
    CROSS JOIN LATERAL (
        SELECT v_theme_names[i] AS name, v_theme_slugs[i] AS slug
        FROM (SELECT 1 + floor(random()*v_theme_len)::int AS i) s
    ) t
    WHERE u.email LIKE '%@duskforge.demo' AND random() < 0.12
    ON CONFLICT (user_id, slug) DO NOTHING;

    -- =========================================================================
    -- 6. REVIEWS — note (0.5 pas, biaisée vers le haut), texte ~85% du temps
    -- =========================================================================
    -- created_at TOUJOURS postérieur à la création du compte de l'auteur
    -- (impossible de noter un film avant d'avoir un compte).
    -- NB : l'index de l'auteur (idx) est tiré DANS la projection de la requête
    -- génératrice (donc une fois par ligne) ; on indexe ensuite uids ET ucreated
    -- avec ce même idx matérialisé → pas de re-tirage, dates cohérentes.
    INSERT INTO reviews (user_id, tmdb_id, rating, content, contains_spoilers, created_at, updated_at)
    SELECT uids[idx], tmdb_id, rating, content, spoiler, t, t
    FROM (
        SELECT idx, tmdb_id, rating, content, spoiler,
               ucreated[idx] + interval '2 hours' + rnd * (NOW() - ucreated[idx] - interval '2 hours') AS t
        FROM (
            SELECT
                1 + floor(random()*v_card)::int AS idx,
                v_mids[1 + floor(random()*v_mcard)::int] AS tmdb_id,
                ((floor(power(random(),0.7)*8) + 3) / 2.0)::numeric(2,1) AS rating,  -- 1.5 .. 5.0, biaisé haut
                CASE WHEN random() < 0.85 THEN
                    v_reviews[1+floor(random()*v_rev_len)::int]
                    || CASE WHEN random() < 0.35 THEN ' ' || v_reviews[1+floor(random()*v_rev_len)::int] ELSE '' END
                ELSE NULL END AS content,
                random() < 0.12 AS spoiler,
                random() AS rnd
            FROM generate_series(1, v_review_attempts) g
        ) base
    ) s
    ON CONFLICT (user_id, tmdb_id) DO NOTHING;

    GET DIAGNOSTICS v_reviews_n = ROW_COUNT;

    -- 6bis. UTILISATEURS "PUISSANTS" : les ~60 premiers comptes (héros + tout
    -- début de cohorte) sont des cinéphiles actifs qui notent une large part du
    -- catalogue → profils riches, succès de critique débloqués, et beaucoup
    -- d'interactions reçues par la suite.
    INSERT INTO reviews (user_id, tmdb_id, rating, content, contains_spoilers, created_at, updated_at)
    SELECT user_id, tmdb_id, rating, content, spoiler, t, t
    FROM (
        SELECT u.id AS user_id, m.tmdb_id AS tmdb_id,
            ((floor(power(random(),0.7)*8) + 3) / 2.0)::numeric(2,1) AS rating,
            CASE WHEN random() < 0.9 THEN
                v_reviews[1+floor(random()*v_rev_len)::int]
                || CASE WHEN random() < 0.4 THEN ' ' || v_reviews[1+floor(random()*v_rev_len)::int] ELSE '' END
            ELSE NULL END AS content,
            random() < 0.1 AS spoiler,
            -- créées après l'ouverture du compte (clamp sur u.created_at)
            u.created_at + interval '2 hours' + (random() * (NOW() - u.created_at - interval '2 hours')) AS t
        FROM generate_series(1, 60) AS hs(i)
        JOIN users u ON u.id = uids[hs.i]
        JOIN _seed_movies m ON random() < 0.72   -- ~61 films sur 85 par cinéphile
        OFFSET 0
    ) s
    ON CONFLICT (user_id, tmdb_id) DO NOTHING;

    SELECT count(*) INTO v_reviews_n FROM reviews;
    RAISE NOTICE '[demo seed] % reviews créées.', v_reviews_n;
    -- (la mise en avant des meilleures reviews est calculée à l'étape 11, après les likes)

    -- =========================================================================
    -- 7. FILMS VUS (collection "watched") — réalisme : on a vu ce qu'on note,
    --    plus un échantillon de films vus sans review.
    -- =========================================================================
    -- 7a. chaque film noté est marqué comme vu
    INSERT INTO collection_items (collection_id, tmdb_id, runtime, added_at, metadata)
    SELECT c.id, r.tmdb_id, COALESCE(m.runtime,0)::smallint, r.created_at, '{}'
    FROM reviews r
    JOIN collections c ON c.user_id = r.user_id AND c.slug = 'watched'
    JOIN _seed_movies m ON m.tmdb_id = r.tmdb_id
    ON CONFLICT (collection_id, tmdb_id) DO NOTHING;

    -- 7b. films vus supplémentaires (volume variable par utilisateur)
    INSERT INTO collection_items (collection_id, tmdb_id, runtime, added_at, metadata)
    SELECT c.id, m.tmdb_id, m.runtime::smallint,
           c.created_at + (random() * (NOW() - c.created_at)), '{}'
    FROM collections c
    JOIN _seed_movies m ON TRUE
    -- décision PAR PAIRE (collection, film) : le hash référence m.tmdb_id, donc
    -- le filtre ne peut pas être remonté au scan de la collection (sinon random()
    -- serait évalué une seule fois par collection → tout ou rien). « Intensité »
    -- propre à l'utilisateur via hash(c.id) → chaque watched a un sous-ensemble varié.
    WHERE c.slug = 'watched'
      AND (('x'||substr(md5(c.id::text||':'||m.tmdb_id::text),1,8))::bit(32)::bigint::numeric % 1000 / 1000.0)
          < (0.05 + 0.60 * (('x'||substr(md5(c.id::text),1,8))::bit(32)::bigint::numeric % 1000 / 1000.0))
    ON CONFLICT (collection_id, tmdb_id) DO NOTHING;

    -- 7c. watchlist "to-watch" (volume plus faible)
    INSERT INTO collection_items (collection_id, tmdb_id, runtime, added_at, metadata)
    SELECT c.id, m.tmdb_id, m.runtime::smallint,
           c.created_at + (random() * (NOW() - c.created_at)), '{}'
    FROM collections c
    JOIN _seed_movies m ON TRUE
    WHERE c.slug = 'to-watch'
      AND (('x'||substr(md5(c.id::text||':'||m.tmdb_id::text),1,8))::bit(32)::bigint::numeric % 1000 / 1000.0)
          < (0.02 + 0.18 * (('x'||substr(md5(c.id::text),1,8))::bit(32)::bigint::numeric % 1000 / 1000.0))
    ON CONFLICT (collection_id, tmdb_id) DO NOTHING;

    -- 7d. collections personnalisées
    INSERT INTO collection_items (collection_id, tmdb_id, runtime, added_at, metadata)
    SELECT c.id, m.tmdb_id, m.runtime::smallint,
           c.created_at + (random() * (NOW() - c.created_at)), '{}'
    FROM collections c
    JOIN _seed_movies m ON TRUE
    WHERE c.type = 'custom'
      AND (('x'||substr(md5(c.id::text||':'||m.tmdb_id::text),1,8))::bit(32)::bigint::numeric % 1000 / 1000.0) < 0.18
    ON CONFLICT (collection_id, tmdb_id) DO NOTHING;

    -- INVARIANT : un film VU ne peut pas rester dans la watchlist (le flux de
    -- review de l'API retire le film de "to-watch" quand on le note). On garantit
    -- donc la disjonction watched ∩ to-watch = ∅ pour chaque utilisateur.
    DELETE FROM collection_items tw
    USING collections tc, collections wc, collection_items wci
    WHERE tw.collection_id = tc.id AND tc.slug = 'to-watch'
      AND wc.user_id = tc.user_id AND wc.slug = 'watched'
      AND wci.collection_id = wc.id AND wci.tmdb_id = tw.tmdb_id;

    RAISE NOTICE '[demo seed] collections & films vus peuplés.';

    -- =========================================================================
    -- 8. FOLLOWS — biais de popularité (power) → les héros sont très suivis
    -- =========================================================================
    -- NB : le tirage aléatoire de la cible est calculé dans la PROJECTION de la
    -- sous-requête (donc une fois PAR ligne du fan-out generate_series), avec
    -- OFFSET 0 comme barrière d'optimisation. Le filtre anti-auto-follow porte
    -- ensuite sur la valeur déjà matérialisée (pas de re-tirage).
    -- created_at >= création des DEUX comptes (impossible de suivre quelqu'un
    -- avant que l'un ou l'autre n'existe) : on borne sur GREATEST des deux dates.
    INSERT INTO follows (follower_id, following_id, created_at)
    SELECT s.follower, s.target, lo + (random() * (NOW() - lo))
    FROM (
        SELECT u.id AS follower, u.created_at AS f_created,
               uids[1 + floor(power(random(),2.2)*v_card)::int] AS target
        FROM users u
        -- nb de follows PROPRE À CHAQUE utilisateur : la borne référence u.id
        -- (donc évaluée par ligne, pas une seule fois pour toute la requête),
        -- via un hash → distribution variée et à longue traîne.
        CROSS JOIN LATERAL generate_series(1, floor(power((('x'||substr(md5(u.id::text||'f'),1,8))::bit(32)::bigint % 1000)/1000.0, 1.5) * v_max_follows)::int) g
        WHERE u.email LIKE '%@duskforge.demo'
        OFFSET 0
    ) s
    JOIN users tu ON tu.id = s.target
    CROSS JOIN LATERAL (SELECT GREATEST(s.f_created, tu.created_at) + interval '1 hour' AS lo) z
    WHERE s.target <> s.follower
    ON CONFLICT (follower_id, following_id) DO NOTHING;

    SELECT count(*) INTO v_follows_n FROM follows;
    RAISE NOTICE '[demo seed] % relations de follow créées.', v_follows_n;

    -- =========================================================================
    -- 9. LIKES DE REVIEWS — nombre variable par review → popularité réaliste
    -- =========================================================================
    INSERT INTO review_likes (user_id, review_id, created_at)
    SELECT liker, review_id, ts
    FROM (
        SELECT uids[1 + floor(random()*v_card)::int] AS liker,
               r.id AS review_id, r.user_id AS owner,
               r.created_at + (random() * (NOW() - r.created_at)) AS ts
        FROM reviews r
        -- popularité PROPRE À CHAQUE review (borne corrélée à r.id), fortement
        -- biaisée (exposant 3) → la plupart peu likées, quelques-unes virales.
        CROSS JOIN LATERAL generate_series(1, floor(power((('x'||substr(md5(r.id::text||'lk'),1,8))::bit(32)::bigint % 1000)/1000.0, 3.0) * 50)::int) g
        -- on ne like QUE les reviews qui ont un texte : une review sans contenu
        -- (note seule) est invisible en front, donc pas de like fantôme.
        WHERE r.content IS NOT NULL
        OFFSET 0
    ) s
    WHERE liker <> owner
    ON CONFLICT (user_id, review_id) DO NOTHING;

    SELECT count(*) INTO v_likes_n FROM review_likes;
    RAISE NOTICE '[demo seed] % likes de reviews créés.', v_likes_n;

    -- =========================================================================
    -- 10. COMMENTAIRES + LIKES DE COMMENTAIRES
    -- =========================================================================
    -- auteur (idx) tiré par ligne dans la projection génératrice ; ts >= max(date
    -- de la review, création de l'auteur) ; on ne commente que les reviews visibles.
    INSERT INTO comments (user_id, review_id, content, contains_spoilers, created_at, updated_at)
    SELECT uids[idx], review_id, body, spoiler, ts, ts
    FROM (
        SELECT idx, review_id, body, spoiler,
               (GREATEST(r_created, ucreated[idx]) + interval '5 minutes')
               + rnd * (NOW() - (GREATEST(r_created, ucreated[idx]) + interval '5 minutes')) AS ts
        FROM (
            SELECT 1 + floor(random()*v_card)::int AS idx,
                   r.id AS review_id, r.created_at AS r_created,
                   v_comments[1+floor(random()*v_com_len)::int] AS body,
                   random() < 0.05 AS spoiler,
                   random() AS rnd
            FROM reviews r
            CROSS JOIN LATERAL generate_series(1, floor(power((('x'||substr(md5(r.id::text||'cm'),1,8))::bit(32)::bigint % 1000)/1000.0, 2.3) * (v_max_comments+2))::int) g
            WHERE r.content IS NOT NULL
        ) base
    ) s;

    INSERT INTO comment_likes (user_id, comment_id, created_at)
    SELECT liker, comment_id, ts
    FROM (
        SELECT uids[1 + floor(random()*v_card)::int] AS liker,
               c.id AS comment_id, c.user_id AS owner,
               c.created_at + (random() * (NOW() - c.created_at)) AS ts
        FROM comments c
        CROSS JOIN LATERAL generate_series(1, floor(power((('x'||substr(md5(c.id::text||'cl'),1,8))::bit(32)::bigint % 1000)/1000.0, 2.6) * (v_max_clikes+3))::int) g
        OFFSET 0
    ) s
    WHERE liker <> owner
    ON CONFLICT (user_id, comment_id) DO NOTHING;

    RAISE NOTICE '[demo seed] commentaires & likes de commentaires créés.';

    -- =========================================================================
    -- 11. REVIEWS MISES EN AVANT — top reviews les plus likées
    -- =========================================================================
    UPDATE reviews SET featured_at = created_at + interval '3 days'
    WHERE id IN (
        SELECT r.id
        FROM reviews r
        JOIN (
            SELECT review_id, count(*) AS likes FROM review_likes GROUP BY review_id
        ) lc ON lc.review_id = r.id
        WHERE r.content IS NOT NULL
        ORDER BY lc.likes DESC
        LIMIT 40
    );

    -- =========================================================================
    -- 12. SUCCÈS (user_achievements) — calculés sur les VRAIS compteurs, avec
    --     les MÊMES définitions que le StatsRepository de l'API (review_count,
    --     rating 5.0, films vus, runtime, likes reçus, followers). Garanties :
    --       • on ne débloque que les paliers réellement atteints (threshold<=n) ;
    --       • tous les paliers inférieurs d'une échelle sont donc débloqués
    --         (impossible d'avoir l'or sans le bronze) ;
    --       • unlocked_at >= date de création du compte (jamais avant) ;
    --       • unlocked_at croît avec le palier (sort_order) → un palier supérieur
    --         est toujours débloqué APRÈS le palier inférieur de la même échelle.
    --     L'expression de date est factorisée ci-dessous (u.created_at, a.sort_order).
    --       GREATEST(création+3h, NOW - (560-sort_order)*0.4 jours - jitter)
    -- =========================================================================

    -- reviews écrites
    INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
    SELECT s.user_id, a.id,
        GREATEST(u.created_at + interval '3 hours',
                 NOW() - interval '1 day' * (560 - a.sort_order) * 0.4
                       - interval '1 hour' * ((('x'||substr(md5(u.id::text||a.code),1,8))::bit(32)::bigint % 12)))
    FROM (SELECT user_id, count(*) AS n FROM reviews GROUP BY user_id) s
    JOIN users u ON u.id = s.user_id
    JOIN achievements a ON a.criterion->>'kind' = 'review_count'
                       AND (a.criterion->'params'->>'threshold')::int <= s.n
    ON CONFLICT DO NOTHING;

    -- notes 5 étoiles données
    INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
    SELECT s.user_id, a.id,
        GREATEST(u.created_at + interval '3 hours',
                 NOW() - interval '1 day' * (560 - a.sort_order) * 0.4
                       - interval '1 hour' * ((('x'||substr(md5(u.id::text||a.code),1,8))::bit(32)::bigint % 12)))
    FROM (SELECT user_id, count(*) AS n FROM reviews WHERE rating = 5.0 GROUP BY user_id) s
    JOIN users u ON u.id = s.user_id
    JOIN achievements a ON a.criterion->>'kind' = 'rating_given'
                       AND (a.criterion->'params'->>'threshold')::int <= s.n
    ON CONFLICT DO NOTHING;

    -- films vus (nombre)
    INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
    SELECT s.user_id, a.id,
        GREATEST(u.created_at + interval '3 hours',
                 NOW() - interval '1 day' * (560 - a.sort_order) * 0.4
                       - interval '1 hour' * ((('x'||substr(md5(u.id::text||a.code),1,8))::bit(32)::bigint % 12)))
    FROM (
        SELECT c.user_id, count(*) AS n
        FROM collections c JOIN collection_items ci ON ci.collection_id = c.id
        WHERE c.slug = 'watched' GROUP BY c.user_id
    ) s
    JOIN users u ON u.id = s.user_id
    JOIN achievements a ON a.criterion->>'kind' = 'watched_count'
                       AND (a.criterion->'params'->>'threshold')::int <= s.n
    ON CONFLICT DO NOTHING;

    -- temps de visionnage (minutes)
    INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
    SELECT s.user_id, a.id,
        GREATEST(u.created_at + interval '3 hours',
                 NOW() - interval '1 day' * (560 - a.sort_order) * 0.4
                       - interval '1 hour' * ((('x'||substr(md5(u.id::text||a.code),1,8))::bit(32)::bigint % 12)))
    FROM (
        SELECT c.user_id, COALESCE(sum(ci.runtime),0) AS mins
        FROM collections c JOIN collection_items ci ON ci.collection_id = c.id
        WHERE c.slug = 'watched' GROUP BY c.user_id
    ) s
    JOIN users u ON u.id = s.user_id
    JOIN achievements a ON a.criterion->>'kind' = 'watched_runtime'
                       AND (a.criterion->'params'->>'minutes')::int <= s.mins
    ON CONFLICT DO NOTHING;

    -- likes reçus
    INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
    SELECT s.user_id, a.id,
        GREATEST(u.created_at + interval '3 hours',
                 NOW() - interval '1 day' * (560 - a.sort_order) * 0.4
                       - interval '1 hour' * ((('x'||substr(md5(u.id::text||a.code),1,8))::bit(32)::bigint % 12)))
    FROM (
        SELECT r.user_id, count(*) AS n
        FROM reviews r JOIN review_likes rl ON rl.review_id = r.id
        GROUP BY r.user_id
    ) s
    JOIN users u ON u.id = s.user_id
    JOIN achievements a ON a.criterion->>'kind' = 'likes_received'
                       AND (a.criterion->'params'->>'threshold')::int <= s.n
    ON CONFLICT DO NOTHING;

    -- nombre de followers
    INSERT INTO user_achievements (user_id, achievement_id, unlocked_at)
    SELECT s.following_id, a.id,
        GREATEST(u.created_at + interval '3 hours',
                 NOW() - interval '1 day' * (560 - a.sort_order) * 0.4
                       - interval '1 hour' * ((('x'||substr(md5(u.id::text||a.code),1,8))::bit(32)::bigint % 12)))
    FROM (SELECT following_id, count(*) AS n FROM follows GROUP BY following_id) s
    JOIN users u ON u.id = s.following_id
    JOIN achievements a ON a.criterion->>'kind' = 'followers_count'
                       AND (a.criterion->'params'->>'threshold')::int <= s.n
    ON CONFLICT DO NOTHING;

    RAISE NOTICE '[demo seed] succès débloqués calculés.';

    -- =========================================================================
    -- 13. MESSAGES PRIVÉS — l'API n'autorise l'envoi qu'entre comptes en FOLLOW
    --     MUTUEL (ErrNotMutualFollow). On ne crée donc de conversation qu'entre
    --     paires qui se suivent dans LES DEUX SENS. On prend la direction
    --     canonique (follower_id < following_id) pour ne traiter chaque paire
    --     qu'une fois ; ~7% de ces paires discutent (fil de longueur variable).
    INSERT INTO messages (sender_id, receiver_id, content, read_at, created_at)
    SELECT sender, receiver, body,
           -- lu dans 70% des cas, TOUJOURS après l'envoi (read_at >= created_at)
           CASE WHEN random() < 0.7 THEN created_at + (random() * (NOW() - created_at)) ELSE NULL END,
           created_at
    FROM (
        SELECT
            CASE WHEN g % 2 = 0 THEN f.follower_id ELSE f.following_id END AS sender,
            CASE WHEN g % 2 = 0 THEN f.following_id ELSE f.follower_id END AS receiver,
            v_messages[1+floor(random()*v_msg_len)::int] AS body,
            NOW() - (random() * interval '120 days') AS created_at
        FROM follows f
        CROSS JOIN LATERAL generate_series(1, floor(2 + power((('x'||substr(md5(f.follower_id::text||f.following_id::text||'mc'),1,8))::bit(32)::bigint % 1000)/1000.0, 1.3) * 10)::int) g
        WHERE f.follower_id < f.following_id
          AND EXISTS (SELECT 1 FROM follows fb WHERE fb.follower_id = f.following_id AND fb.following_id = f.follower_id)
          AND (('x'||substr(md5(f.follower_id::text||f.following_id::text||'ms'),1,8))::bit(32)::bigint % 1000)/1000.0 < 0.07
        OFFSET 0
    ) m;

    -- états de conversation (dans les deux sens) pour que les fils s'affichent proprement
    INSERT INTO conversation_states (user_id, other_user_id, created_at, updated_at)
    SELECT u, o, min(ts), max(ts) FROM (
        SELECT sender_id AS u, receiver_id AS o, created_at AS ts FROM messages
        UNION ALL
        SELECT receiver_id AS u, sender_id AS o, created_at AS ts FROM messages
    ) m
    GROUP BY u, o
    ON CONFLICT (user_id, other_user_id) DO NOTHING;

    -- quelques réactions emoji
    INSERT INTO message_reactions (message_id, user_id, emoji, created_at)
    SELECT m.id, m.receiver_id,
           (ARRAY['👍','😂','❤️','🔥','😮','🎬'])[1+floor(random()*6)::int],
           m.created_at + interval '1 hour'
    FROM messages m
    WHERE random() < 0.08
    ON CONFLICT (message_id, user_id, emoji) DO NOTHING;

    RAISE NOTICE '[demo seed] messages & réactions créés.';

    -- =========================================================================
    -- 14. BLOCAGES — quelques utilisateurs en ont bloqué d'autres
    -- =========================================================================
    INSERT INTO user_blocks (blocker_id, blocked_id, created_at)
    SELECT blocker, blocked, NOW() - (random() * interval '200 days')
    FROM (
        SELECT uids[1+floor(random()*v_card)::int] AS blocker,
               uids[1+floor(random()*v_card)::int] AS blocked
        FROM generate_series(1, 200) g
    ) p
    WHERE blocker <> blocked
    ON CONFLICT (blocker_id, blocked_id) DO NOTHING;

    -- =========================================================================
    -- 15. SIGNALEMENTS (modération) — pour la démo du panneau admin.
    --     Garanties de cohérence (comme via l'API/le Front) :
    --       • exactement UNE cible par signalement (user XOR review XOR comment) ;
    --       • JAMAIS d'auto-signalement (le rapporteur n'est pas l'auteur visé) —
    --         le Front ne propose pas « signaler » sur son propre contenu ;
    --       • created_at >= max(date de la cible, création du rapporteur) ;
    --       • si résolu/rejeté : resolved_at >= created_at ET resolver = un ADMIN
    --         (marie/theo = uids[1]/uids[2]) ; si pending : resolved_at/resolver NULL ;
    --       • 'spoiler' réservé aux contenus (review/comment), pas aux profils.
    -- =========================================================================

    -- 15a. signalements de REVIEWS
    INSERT INTO reports (reporter_id, reason, details, status, target_review_id, created_at, resolved_at, resolver_id)
    SELECT uids[idx], reason, details, status, review_id, cre,
           CASE WHEN status <> 'pending' THEN cre + rr * (NOW() - cre) ELSE NULL END,
           CASE WHEN status <> 'pending' THEN uids[1 + floor(random()*2)::int] ELSE NULL END
    FROM (
        SELECT idx, review_id, author, reason, details, status, rr,
               lo + rc * (NOW() - lo) AS cre
        FROM (
            SELECT 1 + floor(random()*v_card)::int AS idx,
                   rv.id AS review_id, rv.user_id AS author, rv.created_at AS r_created,
                   v_reasons[1+floor(random()*5)::int] AS reason,
                   CASE WHEN random() < 0.5 THEN 'Contenu jugé problématique par un utilisateur.' ELSE NULL END AS details,
                   (ARRAY['pending','pending','pending','resolved','dismissed'])[1+floor(random()*5)::int]::report_status_type AS status,
                   random() AS rc, random() AS rr
            FROM (SELECT id, user_id, created_at FROM reviews WHERE content IS NOT NULL ORDER BY random() LIMIT 90) rv
        ) base
        CROSS JOIN LATERAL (SELECT GREATEST(base.r_created, ucreated[base.idx]) + interval '1 hour' AS lo) z
        WHERE uids[base.idx] <> base.author
    ) s;

    -- 15b. signalements de COMMENTAIRES
    INSERT INTO reports (reporter_id, reason, details, status, target_comment_id, created_at, resolved_at, resolver_id)
    SELECT uids[idx], reason, details, status, comment_id, cre,
           CASE WHEN status <> 'pending' THEN cre + rr * (NOW() - cre) ELSE NULL END,
           CASE WHEN status <> 'pending' THEN uids[1 + floor(random()*2)::int] ELSE NULL END
    FROM (
        SELECT idx, comment_id, author, reason, details, status, rr,
               lo + rc * (NOW() - lo) AS cre
        FROM (
            SELECT 1 + floor(random()*v_card)::int AS idx,
                   cm.id AS comment_id, cm.user_id AS author, cm.created_at AS c_created,
                   v_reasons[1+floor(random()*5)::int] AS reason,
                   CASE WHEN random() < 0.5 THEN 'Commentaire jugé problématique.' ELSE NULL END AS details,
                   (ARRAY['pending','pending','pending','resolved','dismissed'])[1+floor(random()*5)::int]::report_status_type AS status,
                   random() AS rc, random() AS rr
            FROM (SELECT id, user_id, created_at FROM comments ORDER BY random() LIMIT 50) cm
        ) base
        CROSS JOIN LATERAL (SELECT GREATEST(base.c_created, ucreated[base.idx]) + interval '1 hour' AS lo) z
        WHERE uids[base.idx] <> base.author
    ) s;

    -- 15c. signalements de PROFILS (sans 'spoiler', sans s'auto-signaler)
    INSERT INTO reports (reporter_id, reason, details, status, target_user_id, created_at, resolved_at, resolver_id)
    SELECT uids[idx], reason, details, status, target, cre,
           CASE WHEN status <> 'pending' THEN cre + rr * (NOW() - cre) ELSE NULL END,
           CASE WHEN status <> 'pending' THEN uids[1 + floor(random()*2)::int] ELSE NULL END
    FROM (
        SELECT idx, target, reason, details, status, rr,
               lo + rc * (NOW() - lo) AS cre
        FROM (
            SELECT 1 + floor(random()*v_card)::int AS idx,
                   tg.id AS target, tg.created_at AS t_created,
                   (ARRAY['spam','harassment','inappropriate','harassment','other'])[1+floor(random()*5)::int]::report_reason AS reason,
                   CASE WHEN random() < 0.4 THEN 'Comportement signalé par un utilisateur.' ELSE NULL END AS details,
                   (ARRAY['pending','pending','pending','resolved','dismissed'])[1+floor(random()*5)::int]::report_status_type AS status,
                   random() AS rc, random() AS rr
            FROM (SELECT id, created_at FROM users WHERE email LIKE '%@duskforge.demo' ORDER BY random() LIMIT 40) tg
        ) base
        CROSS JOIN LATERAL (SELECT GREATEST(base.t_created, ucreated[base.idx]) + interval '1 hour' AS lo) z
        WHERE uids[base.idx] <> base.target
    ) s;

    RAISE NOTICE '[demo seed] signalements créés.';

    -- =========================================================================
    -- 16. NOTIFICATIONS — dérivées des interactions (cohérence garantie)
    -- =========================================================================
    -- nouveau follower
    INSERT INTO notifications (user_id, actor_id, type, read_at, created_at)
    SELECT f.following_id, f.follower_id, 'new_follow',
           CASE WHEN random() < 0.65 THEN f.created_at + interval '2 hours' ELSE NULL END,
           f.created_at
    FROM follows f
    WHERE random() < 0.6;

    -- like sur une review
    INSERT INTO notifications (user_id, actor_id, type, review_id, read_at, created_at)
    SELECT r.user_id, rl.user_id, 'like_review', r.id,
           CASE WHEN random() < 0.65 THEN rl.created_at + interval '3 hours' ELSE NULL END,
           rl.created_at
    FROM review_likes rl
    JOIN reviews r ON r.id = rl.review_id
    WHERE rl.user_id <> r.user_id AND random() < 0.4;

    -- nouveau commentaire
    INSERT INTO notifications (user_id, actor_id, type, comment_id, read_at, created_at)
    SELECT r.user_id, c.user_id, 'new_comment', c.id,
           CASE WHEN random() < 0.6 THEN c.created_at + interval '2 hours' ELSE NULL END,
           c.created_at
    FROM comments c
    JOIN reviews r ON r.id = c.review_id
    WHERE c.user_id <> r.user_id AND random() < 0.7;

    -- like sur un commentaire
    INSERT INTO notifications (user_id, actor_id, type, comment_id, read_at, created_at)
    SELECT c.user_id, cl.user_id, 'like_comment', c.id,
           CASE WHEN random() < 0.6 THEN cl.created_at + interval '4 hours' ELSE NULL END,
           cl.created_at
    FROM comment_likes cl
    JOIN comments c ON c.id = cl.comment_id
    WHERE cl.user_id <> c.user_id AND random() < 0.4;

    -- succès débloqué — comme l'API : UNE notification par déblocage, message =
    -- nom exact du succès (a.name), actor_id NULL, datée au moment du déblocage.
    -- (la majorité sont déjà lues car anciennes ; read_at toujours >= created_at)
    INSERT INTO notifications (user_id, type, achievement_id, message, read_at, created_at)
    SELECT ua.user_id, 'achievement_unlocked', ua.achievement_id,
           a.name,
           CASE WHEN random() < 0.85 THEN ua.unlocked_at + interval '1 hour' * (1 + floor(random()*72)) ELSE NULL END,
           ua.unlocked_at
    FROM user_achievements ua
    JOIN achievements a ON a.id = ua.achievement_id;

    -- message système de bienvenue pour tous
    INSERT INTO notifications (user_id, type, message, read_at, created_at)
    SELECT id, 'system', 'Bienvenue sur DuskForge 🎬 — commencez par noter votre dernier film !',
           CASE WHEN random() < 0.8 THEN created_at + interval '1 day' ELSE NULL END,
           created_at + interval '1 minute'
    FROM users WHERE email LIKE '%@duskforge.demo';

    RAISE NOTICE '[demo seed] notifications créées.';

    -- =========================================================================
    -- 17. ACTIVITÉS (fil d'actualité) — dérivées des interactions
    -- =========================================================================
    -- review créée (toutes)
    INSERT INTO activities (user_id, type, review_id, created_at)
    -- uniquement les reviews avec texte (les notes seules sont invisibles, donc
    -- pas d'item de fil « a écrit une critique » qui pointerait dans le vide)
    SELECT user_id, 'review_created', id, created_at FROM reviews WHERE content IS NOT NULL;

    -- follow
    INSERT INTO activities (user_id, type, target_user_id, created_at)
    SELECT follower_id, 'user_followed', following_id, created_at
    FROM follows WHERE random() < 0.4;

    -- ajout à la watchlist
    INSERT INTO activities (user_id, type, collection_id, tmdb_id, created_at)
    SELECT c.user_id, 'watchlist_item_added', c.id, ci.tmdb_id, ci.added_at
    FROM collection_items ci
    JOIN collections c ON c.id = ci.collection_id
    WHERE c.slug = 'to-watch' AND random() < 0.25;

    -- ajout à une collection
    INSERT INTO activities (user_id, type, collection_id, tmdb_id, created_at)
    SELECT c.user_id, 'collection_item_added', c.id, ci.tmdb_id, ci.added_at
    FROM collection_items ci
    JOIN collections c ON c.id = ci.collection_id
    WHERE c.type = 'custom' AND random() < 0.4;

    -- commentaire créé
    INSERT INTO activities (user_id, type, comment_id, created_at)
    SELECT user_id, 'comment_created', id, created_at
    FROM comments WHERE random() < 0.5;

    -- review likée
    INSERT INTO activities (user_id, type, review_id, created_at)
    SELECT user_id, 'review_liked', review_id, created_at
    FROM review_likes WHERE random() < 0.04;

    RAISE NOTICE '[demo seed] activités créées.';

    -- =========================================================================
    -- 18. SCÉNARIO DE DÉMO — paire « marie.lefilm » ↔ « paul.spectateur »
    --     • paul = utilisateur RÉGULIER (role user), profil déjà actif.
    --     • follow MUTUEL marie↔paul (et paul↔lea, paul↔theo) → l'API autorise
    --       alors l'envoi de messages (ErrNotMutualFollow sinon).
    --     • conversations prêtes, dont un dernier message NON LU pour marie.
    --     • watchlist de marie garnie de films qu'elle n'a PAS encore vus.
    -- =========================================================================
    SELECT id INTO v_marie FROM users WHERE email = 'marie.lefilm@duskforge.demo';
    SELECT id INTO v_paul  FROM users WHERE email = 'paul.spectateur@duskforge.demo';
    SELECT id INTO v_lea   FROM users WHERE email = 'lea.popcorn@duskforge.demo';
    SELECT id INTO v_theo  FROM users WHERE email = 'theo.cinephile@duskforge.demo';

    -- pas de blocage entre les paires de démo (sinon envoi impossible)
    DELETE FROM user_blocks WHERE (blocker_id, blocked_id) IN (
        (v_paul,v_marie),(v_marie,v_paul),(v_paul,v_lea),(v_lea,v_paul),(v_paul,v_theo),(v_theo,v_paul));

    -- follows MUTUELS (dates bien postérieures à la création des comptes)
    INSERT INTO follows (follower_id, following_id, created_at) VALUES
        (v_paul, v_marie, NOW()-interval '150 days'),
        (v_marie, v_paul, NOW()-interval '148 days'),
        (v_paul, v_lea,  NOW()-interval '210 days'),
        (v_lea,  v_paul, NOW()-interval '208 days'),
        (v_paul, v_theo, NOW()-interval '160 days'),
        (v_theo, v_paul, NOW()-interval '158 days')
    ON CONFLICT (follower_id, following_id) DO NOTHING;

    -- notifications "nouveau follower" pour la paire vedette (si pas déjà créées)
    INSERT INTO notifications (user_id, actor_id, type, read_at, created_at)
    SELECT v_marie, v_paul, 'new_follow', NOW()-interval '150 days'+interval '6 hours', NOW()-interval '150 days'
    WHERE NOT EXISTS (SELECT 1 FROM notifications WHERE user_id=v_marie AND actor_id=v_paul AND type='new_follow');
    INSERT INTO notifications (user_id, actor_id, type, read_at, created_at)
    SELECT v_paul, v_marie, 'new_follow', NOW()-interval '148 days'+interval '4 hours', NOW()-interval '148 days'
    WHERE NOT EXISTS (SELECT 1 FROM notifications WHERE user_id=v_paul AND actor_id=v_marie AND type='new_follow');

    -- activités "a suivi" correspondantes
    INSERT INTO activities (user_id, type, target_user_id, created_at)
    SELECT v_paul, 'user_followed', v_marie, NOW()-interval '150 days'
    WHERE NOT EXISTS (SELECT 1 FROM activities WHERE user_id=v_paul AND type='user_followed' AND target_user_id=v_marie);
    INSERT INTO activities (user_id, type, target_user_id, created_at)
    SELECT v_marie, 'user_followed', v_paul, NOW()-interval '148 days'
    WHERE NOT EXISTS (SELECT 1 FROM activities WHERE user_id=v_marie AND type='user_followed' AND target_user_id=v_paul);

    -- conversations (created_at = NOW - d heures ; read_at = +40 min, NULL si non lu)
    INSERT INTO messages (sender_id, receiver_id, content, created_at, read_at)
    SELECT s, r, c, NOW() - (d * interval '1 hour'),
           CASE WHEN rd THEN NOW() - (d * interval '1 hour') + interval '40 minutes' ELSE NULL END
    FROM (VALUES
        -- marie ↔ paul (le dernier message de paul reste NON LU pour marie)
        (v_paul,  v_marie, 'Salut Marie ! J''adore tes critiques, celle sur Whiplash m''a convaincu de le revoir 🔥', 288, true),
        (v_marie, v_paul,  'Merci Paul, ça me touche ! Whiplash c''est un de mes chouchous 😊', 286, true),
        (v_paul,  v_marie, 'Tu me conseilles quoi dans le genre tendu, huis clos ?', 264, true),
        (v_marie, v_paul,  'Prisoners, sans hésiter. Et garde une soirée entière, ça ne lâche pas.', 263, true),
        (v_paul,  v_marie, 'Vu Prisoners hier soir... la fin 🤯 quelle claque', 216, true),
        (v_marie, v_paul,  'Je savais que ça te plairait ! On en reparle de vive voix 😄', 215, true),
        (v_paul,  v_marie, 'Carrément. Dispo pour une reco ce week-end ?', 48, true),
        (v_paul,  v_marie, 'Hâte de voir ton top de l''année d''ailleurs 👀', 6, false),
        -- paul ↔ lea
        (v_lea,   v_paul,  'Hello ! Si t''aimes l''horreur, fonce sur Hereditary 👻', 200, true),
        (v_paul,  v_lea,   'Ajouté à ma watchlist direct, merci !', 198, true),
        (v_lea,   v_paul,  'Alors, ce Hereditary ?', 70, true),
        (v_paul,  v_lea,   'Traumatisé. Dans le bon sens 😂', 69, true),
        -- paul ↔ theo
        (v_paul,  v_theo,  'Theo, ton analyse de Dune était au top 🙌', 150, true),
        (v_theo,  v_paul,  'Merci ! Vivement la suite au ciné.', 148, true),
        (v_theo,  v_paul,  'Au fait, si tu n''as pas vu Arrival, c''est le même réalisateur — à voir.', 20, true)
    ) AS t(s, r, c, d, rd);

    -- états de conversation pour la paire de démo (ouverts dans les deux sens)
    INSERT INTO conversation_states (user_id, other_user_id, created_at, updated_at)
    SELECT a, b, NOW()-interval '210 days', NOW()
    FROM (VALUES (v_paul,v_marie),(v_marie,v_paul),(v_paul,v_lea),(v_lea,v_paul),(v_paul,v_theo),(v_theo,v_paul)) t(a,b)
    ON CONFLICT (user_id, other_user_id) DO NOTHING;

    -- watchlist de marie : films qu'elle n'a NI vus (watched) NI déjà en watchlist ;
    -- chaque ajout génère aussi l'activité "watchlist_item_added" correspondante.
    WITH tw AS (SELECT id FROM collections WHERE user_id = v_marie AND slug = 'to-watch'),
    added AS (
        INSERT INTO collection_items (collection_id, tmdb_id, runtime, added_at, metadata)
        SELECT (SELECT id FROM tw), m.tmdb_id, m.runtime::smallint,
               NOW() - interval '1 hour' - (random() * interval '18 days'), '{}'
        FROM _seed_movies m
        WHERE m.tmdb_id NOT IN (
                SELECT ci.tmdb_id FROM collection_items ci
                JOIN collections w ON w.id = ci.collection_id
                WHERE w.user_id = v_marie AND w.slug = 'watched')
          AND m.tmdb_id NOT IN (SELECT ci.tmdb_id FROM collection_items ci WHERE ci.collection_id = (SELECT id FROM tw))
        ORDER BY random()
        LIMIT 8
        ON CONFLICT (collection_id, tmdb_id) DO NOTHING
        RETURNING collection_id, tmdb_id, added_at
    )
    INSERT INTO activities (user_id, type, collection_id, tmdb_id, created_at)
    SELECT v_marie, 'watchlist_item_added', collection_id, tmdb_id, added_at FROM added;

    -- collection CUSTOM GARANTIE pour marie (la création de liste perso n'est
    -- sinon attribuée qu'à ~12% des comptes au hasard). « Mes films cultes »,
    -- publique, remplie de films qu'elle a VUS et bien notés (cohérent), avec
    -- les activités "collection_item_added" correspondantes.
    WITH col AS (
        INSERT INTO collections (user_id, name, slug, type, visibility, description, created_at, updated_at)
        VALUES (v_marie, 'Mes films cultes', 'mes-films-cultes', 'custom', 'public',
                'Les films qui m''ont marquée à vie — ma sélection perso 🎬', NOW()-interval '300 days', NOW()-interval '4 days')
        ON CONFLICT (user_id, slug) DO UPDATE SET updated_at = EXCLUDED.updated_at
        RETURNING id
    ),
    items AS (
        INSERT INTO collection_items (collection_id, tmdb_id, runtime, added_at, metadata)
        SELECT (SELECT id FROM col), m.tmdb_id, m.runtime::smallint,
               NOW() - interval '290 days' + (random() * interval '280 days'), '{}'
        FROM _seed_movies m
        WHERE m.tmdb_id IN (
            SELECT r.tmdb_id FROM reviews r
            WHERE r.user_id = v_marie AND r.rating >= 4.0
            ORDER BY r.rating DESC, random() LIMIT 12)
        ON CONFLICT (collection_id, tmdb_id) DO NOTHING
        RETURNING collection_id, tmdb_id, added_at
    )
    INSERT INTO activities (user_id, type, collection_id, tmdb_id, created_at)
    SELECT v_marie, 'collection_item_added', collection_id, tmdb_id, added_at FROM items;

    -- collection custom PRIVÉE garantie pour marie (pour démontrer la visibilité
    -- privée : visible par elle seule, masquée du fil public — le repo d'activités
    -- filtre déjà « c.visibility = 'public' »). Films qu'elle a vus, hors cultes.
    WITH cu AS (SELECT id FROM collections WHERE user_id = v_marie AND slug = 'mes-films-cultes'),
    priv AS (
        INSERT INTO collections (user_id, name, slug, type, visibility, description, created_at, updated_at)
        VALUES (v_marie, 'Plaisirs coupables', 'plaisirs-coupables', 'custom', 'private',
                'Mes petits plaisirs assumés... ou pas 🙈', NOW()-interval '260 days', NOW()-interval '8 days')
        ON CONFLICT (user_id, slug) DO UPDATE SET updated_at = EXCLUDED.updated_at
        RETURNING id
    ),
    items AS (
        INSERT INTO collection_items (collection_id, tmdb_id, runtime, added_at, metadata)
        SELECT (SELECT id FROM priv), m.tmdb_id, m.runtime::smallint,
               NOW() - interval '250 days' + (random() * interval '240 days'), '{}'
        FROM _seed_movies m
        WHERE m.tmdb_id IN (
                SELECT ci.tmdb_id FROM collection_items ci
                JOIN collections w ON w.id = ci.collection_id
                WHERE w.user_id = v_marie AND w.slug = 'watched')
          AND m.tmdb_id NOT IN (SELECT ci.tmdb_id FROM collection_items ci WHERE ci.collection_id = (SELECT id FROM cu))
        ORDER BY random()
        LIMIT 8
        ON CONFLICT (collection_id, tmdb_id) DO NOTHING
        RETURNING collection_id, tmdb_id, added_at
    )
    INSERT INTO activities (user_id, type, collection_id, tmdb_id, created_at)
    SELECT v_marie, 'collection_item_added', collection_id, tmdb_id, added_at FROM items;

    RAISE NOTICE '[demo seed] scénario de démo câblé (marie ↔ paul.spectateur).';

    RAISE NOTICE '[demo seed] ✅ terminé — % utilisateurs, % reviews, % likes, % follows.',
        v_card, v_reviews_n, v_likes_n, v_follows_n;
END
$seed$;
