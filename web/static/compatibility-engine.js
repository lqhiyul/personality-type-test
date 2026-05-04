(function () {
  const CONTEXTS = ["friendship", "relationship", "work"];
  const AXES = [
    { key: "energy", left: "E", right: "I" },
    { key: "information", left: "S", right: "N" },
    { key: "decision", left: "T", right: "F" },
    { key: "rhythm", left: "J", right: "P" },
  ];

  const baseScores = { friendship: 60, relationship: 58, work: 62 };
  const axisScores = {
    energy: {
      same: { friendship: 5, relationship: 3, work: 1 },
      different: { friendship: 1, relationship: -1, work: 2 },
    },
    information: {
      same: { friendship: 5, relationship: 4, work: 5 },
      different: { friendship: 2, relationship: 1, work: 2 },
    },
    decision: {
      same: { friendship: 3, relationship: 2, work: 3 },
      different: { friendship: 3, relationship: 2, work: 4 },
    },
    rhythm: {
      same: { friendship: 4, relationship: 3, work: 4 },
      different: { friendship: 2, relationship: 0, work: 1 },
    },
  };

  const text = {
    uk: {
      score: "потенціалу взаємодії",
      contexts: { friendship: "Дружба", relationship: "Стосунки", work: "Робота" },
      sameType: "Однаковий тип дає швидке взаєморозуміння, але також може підсилювати спільні сліпі зони.",
      lead: {
        friendship: "{pair} у дружбі може триматися на сильній комбінації: {bridge}. Тут важливо не вимагати однакового темпу: краще помічати, де різниця додає простору, а де забирає легкість.",
        relationship: "{pair} у близьких стосунках має потенціал, якщо не плутати різницю стилів із байдужістю. Найкраще ця взаємодія працює, коли пара бачить свою комбінацію: {bridge}, і чесно домовляється про контакт, простір та очікування.",
        work: "{pair} у роботі може бути сильною зв'язкою для задач, де важлива така комбінація: {bridge}. Потенціал розкривається, коли ролі, темп і спосіб прийняття рішень проговорені до старту."
      },
      bridge: {
        sameInformationN: "спільний інтерес до сенсу, можливостей і довгої логіки",
        sameInformationS: "спільну увагу до фактів, конкретики й реального стану справ",
        diffInformation: "поєднання бачення можливостей із перевіркою реальності",
        sameDecisionT: "прямість, критерії й готовність називати проблему",
        sameDecisionF: "увагу до тону, довіри й людського контексту",
        diffDecision: "баланс ясних критеріїв і людського тону",
        sameRhythmJ: "передбачуваність, завершення й повагу до структури",
        sameRhythmP: "гнучкість, живу адаптацію й простір для зміни курсу",
        diffRhythm: "обмін між структурою та адаптацією",
        sameEnergyE: "активний обмін і швидке оживлення контакту",
        sameEnergyI: "спокійний ритм, глибину й повагу до внутрішнього простору",
        diffEnergy: "поєднання зовнішнього контакту з внутрішньою глибиною"
      },
      works: {
        sameType: "Вони швидко впізнають логіку одне одного й менше витрачають сил на переклад базових реакцій.",
        energySame: "Соціальний ритм ближчий, тому легше домовлятися про кількість контакту й пауз.",
        energyDiff: "Один може оживляти контакт назовні, інший - тримати глибину й не давати взаємодії розсипатися.",
        informationSame: "Вони дивляться на реальність схожим способом, тому швидше ловлять, що саме важливо.",
        informationDiff: "Один краще тримає конкретику, інший - ширший горизонт; разом це зменшує ризик однобокості.",
        decisionSame: "У рішеннях легше зрозуміти, що для іншого є переконливим аргументом.",
        decisionDiff: "Різниця між критеріями й людським тоном може робити висновки точнішими.",
        rhythmSame: "Схожий темп завершення допомагає не сперечатися про сам спосіб руху.",
        rhythmDiff: "Структура одного й гнучкість іншого можуть добре доповнюватися, якщо це не перетворюється на контроль.",
        typeStrength: "{a} частіше приносить: {aStrength}. {b} додає: {bStrength}."
      },
      tensions: {
        sameType: "Є ризик одночасно не помічати ту саму слабку ділянку й підкріплювати спільну звичку.",
        sameMirror: "Схожість може давати комфорт, але іноді зменшує кількість зовнішньої корекції.",
        sameComfort: "Якщо обидва уникають одного й того самого складного місця, проблема довше лишається без назви.",
        energySame: "Схожий ритм може здаватися очевидним, тому домовленості про паузи й контакт іноді не проговорюють.",
        informationSame: "Однакова оптика допомагає швидко домовлятися, але може залишати поза увагою інший тип даних.",
        decisionSame: "Схожий стиль рішень може підсилювати тон, якщо обидва впевнені, що їхній критерій уже достатній.",
        rhythmSame: "Схожий темп руху допомагає, але іноді закріплює спільний спосіб відкладати або форсувати рішення.",
        energyDiff: "Напруга з'являється, коли потребу в паузі читають як холодність, а потребу в контакті - як тиск.",
        informationDiff: "Можуть сперечатися не про висновок, а про те, що взагалі вважати важливими даними.",
        decisionDiff: "Один може чути суху логіку, інший - надмірну м'якість або обхід прямої відповіді.",
        rhythmDiff: "Конфлікт часто виникає між бажанням зафіксувати план і потребою лишити простір для зміни.",
        friendship: "У дружбі складність починається там, де різницю ритму сприймають як особисте ставлення.",
        relationship: "У стосунках важливо не накопичувати дрібні очікування мовчки: саме вони швидко стають більшими за проблему.",
        work: "У роботі ризик у тому, що один бере на себе структуру, а інший адаптацію, але домовленість про відповідальність лишається нечіткою."
      },
      tips: {
        friendship: ["Домовляйтеся про комфортну частоту контакту без перевірок на лояльність.", "Залишайте різниці право бути різницею, а не ознакою байдужості.", "Краще проговорити межу рано, ніж чекати, поки накопичиться роздратування."],
        relationship: ["Розділяйте потребу в просторі й відсутність інтересу: це не одне й те саме.", "Говоріть про очікування конкретно, без тестів і натяків.", "У конфлікті спершу назвіть суть, а вже потім шукайте форму, яка збереже контакт."],
        work: ["На старті зафіксуйте ролі, дедлайни й формат рішень.", "Не змушуйте всіх працювати в одному темпі: важливіше синхронізувати контрольні точки.", "Після напружених рішень коротко перевіряйте, що команда однаково зрозуміла наступний крок."]
      },
      scales: {
        friendship: ["Комунікація", "Легкість контакту", "Довіра", "Спільний ритм"],
        relationship: ["Емоційний комфорт", "Межі", "Розмова про конфлікт", "Побутовий ритм"],
        work: ["Комунікація", "Рішення", "Структура", "Адаптація"]
      },
      copy: "{pair} · {context} — {score}% {scoreLabel}. {summary}"
    },
    ru: {
      score: "потенциала взаимодействия",
      contexts: { friendship: "Дружба", relationship: "Отношения", work: "Работа" },
      sameType: "Одинаковый тип дает быстрое взаимопонимание, но может усиливать общие слепые зоны.",
      lead: {
        friendship: "{pair} в дружбе может держаться на сильной комбинации: {bridge}. Здесь важно не требовать одинакового темпа: лучше замечать, где разница добавляет воздуха, а где забирает легкость.",
        relationship: "{pair} в близких отношениях имеет потенциал, если не путать разницу стилей с равнодушием. Лучше всего такая динамика работает, когда пара видит свою комбинацию: {bridge}, и честно договаривается о контакте, пространстве и ожиданиях.",
        work: "{pair} в работе может быть сильной связкой для задач, где важна такая комбинация: {bridge}. Потенциал раскрывается, когда роли, темп и способ принятия решений проговорены до старта."
      },
      bridge: {
        sameInformationN: "общий интерес к смыслу, возможностям и длинной логике",
        sameInformationS: "общее внимание к фактам, конкретике и реальному положению дел",
        diffInformation: "соединение видения возможностей с проверкой реальности",
        sameDecisionT: "прямота, критерии и готовность назвать проблему",
        sameDecisionF: "внимание к тону, доверию и человеческому контексту",
        diffDecision: "баланс ясных критериев и человеческого тона",
        sameRhythmJ: "предсказуемость, завершение и уважение к структуре",
        sameRhythmP: "гибкость, живую адаптацию и место для смены курса",
        diffRhythm: "обмен между структурой и адаптацией",
        sameEnergyE: "активный обмен и быстрое оживление контакта",
        sameEnergyI: "спокойный ритм, глубину и уважение к внутреннему пространству",
        diffEnergy: "соединение внешнего контакта с внутренней глубиной"
      },
      works: {
        sameType: "Они быстро узнают логику друг друга и меньше тратят сил на перевод базовых реакций.",
        energySame: "Социальный ритм ближе, поэтому проще договариваться о количестве контакта и пауз.",
        energyDiff: "Один может оживлять контакт вовне, другой - держать глубину и не давать взаимодействию рассыпаться.",
        informationSame: "Они смотрят на реальность похожим способом, поэтому быстрее ловят, что действительно важно.",
        informationDiff: "Один лучше держит конкретику, другой - широкий горизонт; вместе это снижает риск однобокости.",
        decisionSame: "В решениях легче понять, что для другого является убедительным аргументом.",
        decisionDiff: "Разница между критериями и человеческим тоном может делать выводы точнее.",
        rhythmSame: "Похожий темп завершения помогает не спорить о самом способе движения.",
        rhythmDiff: "Структура одного и гибкость другого могут хорошо дополнять друг друга, если это не превращается в контроль.",
        typeStrength: "{a} чаще приносит: {aStrength}. {b} добавляет: {bStrength}."
      },
      tensions: {
        sameType: "Есть риск одновременно не замечать одну и ту же слабую зону и подкреплять общую привычку.",
        sameMirror: "Сходство дает комфорт, но иногда уменьшает количество внешней коррекции.",
        sameComfort: "Если оба обходят одну и ту же сложную зону, проблема дольше остается без названия.",
        energySame: "Похожий ритм кажется очевидным, поэтому договоренности о паузах и контакте иногда не проговаривают.",
        informationSame: "Одинаковая оптика помогает быстро договориться, но может оставлять вне внимания другой тип данных.",
        decisionSame: "Похожий стиль решений может усиливать тон, если оба уверены, что их критерий уже достаточен.",
        rhythmSame: "Похожий темп движения помогает, но иногда закрепляет общий способ откладывать или форсировать решения.",
        energyDiff: "Напряжение появляется, когда потребность в паузе читают как холодность, а потребность в контакте - как давление.",
        informationDiff: "Они могут спорить не о выводе, а о том, что вообще считать важными данными.",
        decisionDiff: "Один может слышать сухую логику, другой - чрезмерную мягкость или уход от прямого ответа.",
        rhythmDiff: "Конфликт часто возникает между желанием зафиксировать план и потребностью оставить место для изменений.",
        friendship: "В дружбе сложность начинается там, где разницу ритма воспринимают как личное отношение.",
        relationship: "В отношениях важно не копить мелкие ожидания молча: именно они быстро становятся больше самой проблемы.",
        work: "В работе риск в том, что один берет на себя структуру, другой адаптацию, но договоренность об ответственности остается нечеткой."
      },
      tips: {
        friendship: ["Договаривайтесь о комфортной частоте контакта без проверок на лояльность.", "Оставляйте разнице право быть разницей, а не признаком равнодушия.", "Лучше обозначить границу рано, чем ждать, пока накопится раздражение."],
        relationship: ["Разделяйте потребность в пространстве и отсутствие интереса: это не одно и то же.", "Говорите об ожиданиях конкретно, без тестов и намеков.", "В конфликте сначала назовите суть, а затем ищите форму, которая сохранит контакт."],
        work: ["На старте зафиксируйте роли, дедлайны и формат решений.", "Не заставляйте всех работать в одном темпе: важнее синхронизировать контрольные точки.", "После напряженных решений коротко проверяйте, что команда одинаково поняла следующий шаг."]
      },
      scales: {
        friendship: ["Коммуникация", "Легкость контакта", "Доверие", "Общий ритм"],
        relationship: ["Эмоциональный комфорт", "Границы", "Разговор о конфликте", "Бытовой ритм"],
        work: ["Коммуникация", "Решения", "Структура", "Адаптация"]
      },
      copy: "{pair} · {context} — {score}% {scoreLabel}. {summary}"
    },
    en: {
      score: "interaction potential",
      contexts: { friendship: "Friendship", relationship: "Relationship", work: "Work" },
      sameType: "The same type often brings quick mutual understanding, but it can also amplify the same blind spots.",
      lead: {
        friendship: "{pair} in friendship can work through a strong combination: {bridge}. The key is not forcing the same pace: notice where difference gives the friendship room, and where it starts to drain ease.",
        relationship: "{pair} in a close relationship has potential when different styles are not mistaken for lack of care. This dynamic works best when the pair understands its combination: {bridge}, and agrees clearly on contact, space, and expectations.",
        work: "{pair} at work can be a strong pairing for tasks where this combination matters: {bridge}. The potential shows up when roles, pace, and decision rules are clear before the work starts."
      },
      bridge: {
        sameInformationN: "shared interest in meaning, possibilities, and long-range logic",
        sameInformationS: "shared attention to facts, concrete details, and what is actually happening",
        diffInformation: "a mix of possibility-seeing and reality-checking",
        sameDecisionT: "directness, criteria, and willingness to name the problem",
        sameDecisionF: "attention to tone, trust, and human context",
        diffDecision: "a balance of clear criteria and human tone",
        sameRhythmJ: "predictability, follow-through, and respect for structure",
        sameRhythmP: "flexibility, live adaptation, and room to change course",
        diffRhythm: "an exchange between structure and adaptation",
        sameEnergyE: "active exchange and quick social momentum",
        sameEnergyI: "a calmer rhythm, depth, and respect for inner space",
        diffEnergy: "a mix of outward contact and inner depth"
      },
      works: {
        sameType: "They recognize each other's logic quickly and spend less energy translating basic reactions.",
        energySame: "Their social rhythm is closer, so it is easier to agree on contact and pauses.",
        energyDiff: "One can bring contact outward while the other keeps depth and prevents the interaction from scattering.",
        informationSame: "They read reality in a similar way, so they catch what matters faster.",
        informationDiff: "One keeps the concrete ground, the other keeps the wider horizon; together this reduces one-sidedness.",
        decisionSame: "In decisions, it is easier to understand what counts as a convincing argument for the other person.",
        decisionDiff: "The contrast between criteria and human tone can make conclusions more precise.",
        rhythmSame: "A similar pace of closure helps them avoid arguing about the basic way of moving forward.",
        rhythmDiff: "One person's structure and the other's flexibility can complement each other when it does not become control.",
        typeStrength: "{a} often brings: {aStrength}. {b} adds: {bStrength}."
      },
      tensions: {
        sameType: "They may miss the same weak spot at the same time and reinforce a shared habit.",
        sameMirror: "Similarity can feel comfortable, but it may reduce the amount of outside correction.",
        sameComfort: "If both avoid the same difficult area, the real issue can stay unnamed for longer.",
        energySame: "A similar rhythm can feel obvious, so agreements about contact and pauses may stay unspoken.",
        informationSame: "A shared lens helps them align quickly, but it can leave another kind of data outside the frame.",
        decisionSame: "A similar decision style can intensify the tone when both assume their criterion is already enough.",
        rhythmSame: "A similar pace helps, but it can also reinforce a shared habit of delaying or forcing decisions.",
        energyDiff: "Tension appears when a need for pause is read as coldness, or a need for contact is read as pressure.",
        informationDiff: "They may argue not about the conclusion, but about what should count as important data.",
        decisionDiff: "One may hear dry logic; the other may hear too much softening or avoidance of a direct answer.",
        rhythmDiff: "Conflict often sits between the wish to fix the plan and the need to keep room for change.",
        friendship: "In friendship, friction starts when a difference in rhythm is taken as a personal signal.",
        relationship: "In relationships, small unspoken expectations matter: they can grow larger than the original issue.",
        work: "At work, the risk is that one person carries structure and the other adapts, while ownership stays unclear."
      },
      tips: {
        friendship: ["Agree on a comfortable amount of contact without testing loyalty.", "Let difference remain difference instead of reading it as lack of care.", "Name a boundary early rather than waiting until irritation builds up."],
        relationship: ["Separate the need for space from lack of interest: they are not the same thing.", "Talk about expectations concretely, without tests or hints.", "In conflict, name the issue first, then choose a form that keeps contact intact."],
        work: ["Set roles, deadlines, and decision rules before the work starts.", "Do not force everyone into the same working pace; synchronize checkpoints instead.", "After tense decisions, briefly check that everyone understands the next step the same way."]
      },
      scales: {
        friendship: ["Communication", "Ease", "Trust", "Shared rhythm"],
        relationship: ["Emotional comfort", "Boundaries", "Conflict talk", "Daily rhythm"],
        work: ["Communication", "Decisions", "Structure", "Adaptation"]
      },
      copy: "{pair} · {context} — {score}% {scoreLabel}. {summary}"
    }
  };

  const curatedPairBoosts = {
    "ENFJ|INTJ": { friendship: 9, relationship: 20, work: 8 },
  };

  const curatedPairCopy = {
    uk: {
      "ENFJ|INTJ": {
        friendship: {
          summary: "INTJ і ENFJ у дружбі можуть давати одне одному рідкісний баланс: глибину, напрям і теплий людський контакт. Найкраще це працює, коли INTJ не ховає довіру за дистанцією, а ENFJ не намагається витягнути близькість швидше, ніж вона дозріває.",
          works: ["INTJ додає структуру, чесність і бачення наслідків.", "ENFJ приносить тепло, соціальний контекст і відчуття спільного руху.", "Різниця темпу може робити дружбу живою, якщо її не читати як байдужість або тиск."],
          tensions: ["INTJ може закриватися саме тоді, коли ENFJ хоче більше відкритого контакту.", "ENFJ може брати на себе забагато емоційної роботи й чекати швидшої відповіді.", "Обом важливо не плутати мовчання з холодністю, а турботу з контролем."],
          tips: ["Домовляйтеся про простір і частоту контакту без перевірок на лояльність.", "INTJ варто прямо називати, коли довіра є, навіть якщо емоцій мало назовні.", "ENFJ краще питати, а не вгадувати стан INTJ за паузами."],
        },
        relationship: {
          summary: "У близьких стосунках INTJ і ENFJ мають сильний потенціал через поєднання стратегічності та емоційної навігації. Напруга з'являється не через різницю як таку, а коли простір INTJ сприймається як віддалення, а тепло ENFJ - як тиск.",
          works: ["INTJ допомагає стосункам не втрачати напрям, чесність і реалістичність.", "ENFJ підтримує живий контакт і не дає важливому перетворитися лише на логіку.", "Пара добре росте, коли говорить про очікування до того, як вони стають образою."],
          tensions: ["INTJ може різко обрізати емоційний шар, якщо він здається нечітким.", "ENFJ може занадто швидко просити відкритості там, де INTJ ще збирає думки.", "Конфлікт посилюється, якщо один іде в тишу, а інший починає активніше шукати реакцію."],
          tips: ["Розділяйте потребу в просторі й втрату інтересу: це різні речі.", "Називайте важливі почуття простими словами, без тестів і натяків.", "У складній розмові тримайте і суть, і тон: цій парі потрібні обидва."],
        },
        work: {
          summary: "У роботі INTJ і ENFJ можуть бути дуже сильною зв'язкою: INTJ бачить систему, ризики й довгу логіку, ENFJ переводить це в людський контекст і рух команди. Найкраще пара працює, коли стратегія й комунікація не конкурують, а підтримують одна одну.",
          works: ["INTJ дає архітектуру рішення, пріоритети й тверезу оцінку ризиків.", "ENFJ пояснює ідею людям, тримає включення команди й атмосферу руху.", "Разом вони можуть поєднати сенс, структуру й здатність довести задум до людей."],
          tensions: ["INTJ може занадто довго лишатися в моделі, не показуючи логіку команді.", "ENFJ може взяти на себе забагато мотивації, узгодження й людського стану.", "Напруга виникає, якщо рішення приймаються без чіткої межі між стратегією та комунікацією."],
          tips: ["На старті розділіть ролі: хто тримає модель, хто комунікацію, хто фінальне рішення.", "INTJ варто раніше показувати хід думки, а не тільки готовий висновок.", "ENFJ варто не пом'якшувати ризики, а допомагати команді зрозуміти їх без паніки."],
        },
      },
    },
    ru: {
      "ENFJ|INTJ": {
        friendship: {
          summary: "INTJ и ENFJ в дружбе могут давать друг другу редкий баланс: глубину, направление и теплый человеческий контакт. Лучше всего это работает, когда INTJ не прячет доверие за дистанцией, а ENFJ не пытается вытянуть близость быстрее, чем она созревает.",
          works: ["INTJ добавляет структуру, честность и видение последствий.", "ENFJ приносит тепло, социальный контекст и ощущение общего движения.", "Разный темп может делать дружбу живой, если не читать его как равнодушие или давление."],
          tensions: ["INTJ может закрываться именно тогда, когда ENFJ хочет больше открытого контакта.", "ENFJ может брать на себя слишком много эмоциональной работы и ждать более быстрой реакции.", "Обоим важно не путать молчание с холодностью, а заботу с контролем."],
          tips: ["Договаривайтесь о пространстве и частоте контакта без проверок на лояльность.", "INTJ стоит прямо обозначать, что доверие есть, даже если эмоций мало снаружи.", "ENFJ лучше спрашивать, а не угадывать состояние INTJ по паузам."],
        },
        relationship: {
          summary: "В близких отношениях INTJ и ENFJ имеют сильный потенциал за счет сочетания стратегичности и эмоциональной навигации. Напряжение появляется не из-за разницы самой по себе, а когда пространство INTJ читается как отдаление, а тепло ENFJ - как давление.",
          works: ["INTJ помогает отношениям не терять направление, честность и реалистичность.", "ENFJ поддерживает живой контакт и не дает важному превратиться только в логику.", "Пара хорошо растет, когда говорит об ожиданиях до того, как они становятся обидой."],
          tensions: ["INTJ может резко обрезать эмоциональный слой, если он кажется неясным.", "ENFJ может слишком быстро просить открытости там, где INTJ еще собирает мысли.", "Конфликт усиливается, если один уходит в тишину, а другой начинает активнее искать реакцию."],
          tips: ["Разделяйте потребность в пространстве и потерю интереса: это разные вещи.", "Называйте важные чувства простыми словами, без тестов и намеков.", "В сложном разговоре держите и суть, и тон: этой паре нужны оба уровня."],
        },
        work: {
          summary: "В работе INTJ и ENFJ могут быть очень сильной связкой: INTJ видит систему, риски и длинную логику, ENFJ переводит это в человеческий контекст и движение команды. Лучше всего пара работает, когда стратегия и коммуникация не конкурируют, а поддерживают друг друга.",
          works: ["INTJ дает архитектуру решения, приоритеты и трезвую оценку рисков.", "ENFJ объясняет идею людям, держит вовлеченность команды и атмосферу движения.", "Вместе они могут соединить смысл, структуру и способность довести замысел до людей."],
          tensions: ["INTJ может слишком долго оставаться в модели, не показывая логику команде.", "ENFJ может взять на себя слишком много мотивации, согласования и человеческого состояния.", "Напряжение возникает, если решения принимаются без четкой границы между стратегией и коммуникацией."],
          tips: ["На старте разделите роли: кто держит модель, кто коммуникацию, кто финальное решение.", "INTJ стоит раньше показывать ход мысли, а не только готовый вывод.", "ENFJ стоит не смягчать риски, а помогать команде понять их без паники."],
        },
      },
    },
    en: {
      "ENFJ|INTJ": {
        friendship: {
          summary: "INTJ and ENFJ can make a strong friendship when depth, direction, and warm human contact are allowed to coexist. It works best when INTJ does not hide trust behind distance, and ENFJ does not pull for closeness faster than it is ready to grow.",
          works: ["INTJ brings structure, honesty, and a clear sense of consequences.", "ENFJ adds warmth, social context, and a feeling of shared movement.", "The difference in pace can keep the friendship alive when it is not read as indifference or pressure."],
          tensions: ["INTJ may close down just when ENFJ wants more visible contact.", "ENFJ may take on too much emotional labor and expect a faster response.", "Both need to avoid mistaking silence for coldness, or care for control."],
          tips: ["Agree on space and contact rhythm without turning it into a loyalty test.", "INTJ should name trust directly, even when little emotion shows outwardly.", "ENFJ should ask instead of reading INTJ's pauses as a full answer."],
        },
        relationship: {
          summary: "In close relationships, INTJ and ENFJ have strong potential through the mix of strategy and emotional navigation. The tension is not the difference itself, but what happens when INTJ's need for space is read as withdrawal and ENFJ's warmth is felt as pressure.",
          works: ["INTJ helps the relationship keep direction, honesty, and realism.", "ENFJ keeps the contact alive and prevents important things from becoming only logic.", "The pair grows well when expectations are named before they turn into hurt."],
          tensions: ["INTJ may cut off the emotional layer when it feels too vague.", "ENFJ may ask for openness faster than INTJ can organize the thought.", "Conflict escalates when one person goes quiet and the other pushes harder for a reaction."],
          tips: ["Separate the need for space from loss of interest: they are not the same thing.", "Name important feelings plainly, without tests or hints.", "In difficult talks, keep both substance and tone; this pair needs both."],
        },
        work: {
          summary: "At work, INTJ and ENFJ can be a very strong pairing: INTJ sees the system, risks, and long logic, while ENFJ translates that into human context and team movement. The pair works best when strategy and communication support each other instead of competing.",
          works: ["INTJ gives the architecture of the decision, priorities, and a sober view of risk.", "ENFJ explains the idea to people, keeps the team engaged, and maintains forward motion.", "Together they can connect meaning, structure, and the ability to bring an idea to people."],
          tensions: ["INTJ may stay inside the model too long without showing the reasoning to the team.", "ENFJ may take on too much motivation, alignment, and emotional maintenance.", "Tension appears when decisions lack a clear boundary between strategy and communication."],
          tips: ["At the start, split roles: who holds the model, who handles communication, and who makes the final call.", "INTJ should show the reasoning earlier, not only the finished conclusion.", "ENFJ should not soften risks away, but help the team understand them without panic."],
        },
      },
    },
  };

  function clamp(value, min, max) {
    return Math.max(min, Math.min(max, value));
  }

  function pairKey(aCode, bCode) {
    return [aCode, bCode].sort().join("|");
  }

  function curatedPair(locale, aCode, bCode, context) {
    return curatedPairCopy[locale]?.[pairKey(aCode, bCode)]?.[context] || null;
  }

  function interpolate(template, params) {
    return String(template || "").replace(/\{(\w+)\}/g, (_, key) => String(params[key] ?? ""));
  }

  function letters(code) {
    const value = String(code || "").toUpperCase();
    return { energy: value[0], information: value[1], decision: value[2], rhythm: value[3] };
  }

  function safeLang(lang) {
    return text[lang] ? lang : "uk";
  }

  function typeLabel(type) {
    return `${type.code} (${type.name || type.code})`;
  }

  function axisSame(a, b, axis) {
    return a[axis] === b[axis];
  }

  function bridgeKey(axis, same, valueA, valueB) {
    if (!same) {
      if (axis === "information") return "diffInformation";
      if (axis === "decision") return "diffDecision";
      if (axis === "rhythm") return "diffRhythm";
      return "diffEnergy";
    }
    if (axis === "information") return valueA === "N" ? "sameInformationN" : "sameInformationS";
    if (axis === "decision") return valueA === "T" ? "sameDecisionT" : "sameDecisionF";
    if (axis === "rhythm") return valueA === "J" ? "sameRhythmJ" : "sameRhythmP";
    return valueA === "E" ? "sameEnergyE" : "sameEnergyI";
  }

  function bridgeFor(locale, aLetters, bLetters, context) {
    const primaryAxis = context === "work" ? "information" : context === "relationship" ? "decision" : "energy";
    const secondaryAxis = context === "work" ? "decision" : context === "relationship" ? "energy" : "information";
    const primary = text[locale].bridge[bridgeKey(primaryAxis, axisSame(aLetters, bLetters, primaryAxis), aLetters[primaryAxis], bLetters[primaryAxis])];
    const secondary = text[locale].bridge[bridgeKey(secondaryAxis, axisSame(aLetters, bLetters, secondaryAxis), aLetters[secondaryAxis], bLetters[secondaryAxis])];
    return `${primary}; ${secondary}`;
  }

  function scorePair(aLetters, bLetters, context, aCode = "", bCode = "") {
    let score = baseScores[context] || 60;
    AXES.forEach((axis) => {
      const same = axisSame(aLetters, bLetters, axis.key);
      score += axisScores[axis.key][same ? "same" : "different"][context];
    });
    if (Object.keys(aLetters).every((key) => aLetters[key] === bLetters[key])) score += context === "relationship" ? 6 : 7;
    if (context === "work" && !axisSame(aLetters, bLetters, "decision")) score += 2;
    if (context === "relationship" && !axisSame(aLetters, bLetters, "energy")) score -= 1;
    score += curatedPairBoosts[pairKey(aCode, bCode)]?.[context] || 0;
    return clamp(Math.round(score), 45, 96);
  }

  function unique(items) {
    return [...new Set(items.filter(Boolean))];
  }

  function firstStrength(type) {
    return type?.summary?.strengths?.[0] || type?.tagline || type?.name || type?.code;
  }

  function works(locale, a, b, aLetters, bLetters) {
    const sameType = a.code === b.code;
    return unique([
      sameType ? text[locale].works.sameType : "",
      text[locale].works[axisSame(aLetters, bLetters, "energy") ? "energySame" : "energyDiff"],
      text[locale].works[axisSame(aLetters, bLetters, "information") ? "informationSame" : "informationDiff"],
      text[locale].works[axisSame(aLetters, bLetters, "decision") ? "decisionSame" : "decisionDiff"],
      text[locale].works[axisSame(aLetters, bLetters, "rhythm") ? "rhythmSame" : "rhythmDiff"],
      interpolate(text[locale].works.typeStrength, {
        a: a.code,
        b: b.code,
        aStrength: firstStrength(a),
        bStrength: firstStrength(b),
      }),
    ]).slice(0, 3);
  }

  function tensions(locale, a, b, aLetters, bLetters, context) {
    const items = [];
    if (a.code === b.code) items.push(text[locale].tensions.sameType, text[locale].tensions.sameMirror, text[locale].tensions.sameComfort);
    if (!axisSame(aLetters, bLetters, "energy")) items.push(text[locale].tensions.energyDiff);
    if (!axisSame(aLetters, bLetters, "information")) items.push(text[locale].tensions.informationDiff);
    if (!axisSame(aLetters, bLetters, "decision")) items.push(text[locale].tensions.decisionDiff);
    if (!axisSame(aLetters, bLetters, "rhythm")) items.push(text[locale].tensions.rhythmDiff);
    if (axisSame(aLetters, bLetters, "energy")) items.push(text[locale].tensions.energySame);
    if (axisSame(aLetters, bLetters, "information")) items.push(text[locale].tensions.informationSame);
    if (axisSame(aLetters, bLetters, "decision")) items.push(text[locale].tensions.decisionSame);
    if (axisSame(aLetters, bLetters, "rhythm")) items.push(text[locale].tensions.rhythmSame);
    items.push(text[locale].tensions[context]);
    return unique(items).slice(0, 3);
  }

  function scaleValue(score, aLetters, bLetters, axisKeys) {
    const adjustment = axisKeys.reduce((sum, key) => sum + (axisSame(aLetters, bLetters, key) ? 4 : -1), 0);
    return clamp(Math.round(score + adjustment - 5), 45, 88);
  }

  function scales(locale, context, score, aLetters, bLetters) {
    const labels = text[locale].scales[context] || text[locale].scales.friendship;
    const axisSets = context === "work"
      ? [["energy", "decision"], ["decision", "rhythm"], ["rhythm", "information"], ["information", "rhythm"]]
      : context === "relationship"
        ? [["energy", "decision"], ["energy", "rhythm"], ["decision"], ["rhythm", "information"]]
        : [["energy", "decision"], ["energy"], ["decision", "information"], ["rhythm"]];
    return labels.map((label, index) => ({
      label,
      value: scaleValue(score, aLetters, bLetters, axisSets[index] || ["energy"]),
    }));
  }

  function analyze(input = {}) {
    const locale = safeLang(input.lang);
    const context = CONTEXTS.includes(input.context) ? input.context : "friendship";
    const a = input.typeA;
    const b = input.typeB;
    if (!a?.code || !b?.code) return null;

    const aLetters = letters(a.code);
    const bLetters = letters(b.code);
    const pair = `${a.code} × ${b.code}`;
    const score = scorePair(aLetters, bLetters, context, a.code, b.code);
    const bridge = bridgeFor(locale, aLetters, bLetters, context);
    const conclusion = interpolate(text[locale].lead[context], { pair, bridge });
    const note = a.code === b.code ? text[locale].sameType : "";
    const curated = curatedPair(locale, a.code, b.code, context);
    const summary = curated?.summary || (note ? `${conclusion} ${note}` : conclusion);
    const result = {
      pair,
      title: `${typeLabel(a)} × ${typeLabel(b)}`,
      context,
      contextLabel: text[locale].contexts[context],
      score,
      scoreLabel: text[locale].score,
      summary,
      works: curated?.works || works(locale, a, b, aLetters, bLetters),
      tensions: curated?.tensions || tensions(locale, a, b, aLetters, bLetters, context),
      tips: curated?.tips || text[locale].tips[context].slice(0, 3),
      scales: scales(locale, context, score, aLetters, bLetters),
      url: input.url || "",
    };
    result.copyText = `${interpolate(text[locale].copy, {
      pair: result.pair,
      context: result.contextLabel,
      score: result.score,
      scoreLabel: result.scoreLabel,
      summary: result.summary,
    })}${result.url ? ` ${result.url}` : ""}`;
    return result;
  }

  window.COMPATIBILITY_ENGINE = { contexts: CONTEXTS, analyze };
})();
