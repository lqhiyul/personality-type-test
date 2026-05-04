(function () {
  const similarAxisOrder = [
    { key: "JP", index: 3, flips: { J: "P", P: "J" } },
    { key: "EI", index: 0, flips: { E: "I", I: "E" } },
    { key: "TF", index: 2, flips: { T: "F", F: "T" } },
    { key: "SN", index: 1, flips: { S: "N", N: "S" } },
  ];

  const text = {
    uk: {
      confidence: {
        title: "Впевненість результату",
        high: "Висока",
        medium: "Середня",
        close: "Близький результат",
        highText: "Результат виглядає досить чітким: більшість шкал мають помітний нахил у свій бік.",
        mediumText: "Результат має робочий напрям, але одна зі шкал близька. Варто читати тип як сильну гіпотезу, а не як остаточний ярлик.",
        closeText: "Результат близький: кілька шкал майже врівноважені. Схожі типи можуть допомогти точніше впізнати свій стиль.",
        closest: "Найближча шкала: {axis} ({left} {leftScore} / {right} {rightScore}).",
      },
      why: {
        title: "Чому саме цей тип",
        intro: "Тип складається з чотирьох невеликих нахилів у відповідях. Тут коротко видно, яка сторона кожної шкали проявилася частіше.",
        axis: {
          E: "Відповіді частіше йдуть у зовнішній обмін: думку легше перевіряти в розмові, дії або живому контакті.",
          I: "Відповіді частіше показують внутрішню обробку: спершу зібрати думку всередині, а вже потім говорити або діяти.",
          S: "Помітніший фокус на фактах, конкретиці й тому, що вже можна перевірити на практиці.",
          N: "Помітніший фокус на зв'язках, сенсах і можливих сценаріях розвитку.",
          T: "Рішення частіше спираються на логіку, критерії й бажання бачити, що саме працює.",
          F: "Рішення частіше враховують людський вплив, тон взаємодії й цінність контакту.",
          J: "Відповіді частіше тягнуться до структури: ясного плану, меж і завершення.",
          P: "Відповіді частіше залишають простір для маневру, адаптації й живого процесу.",
        },
      },
      similar: {
        title: "Схожі типи",
        intro: "Ці типи близькі за трьома шкалами, але відрізняються одним важливим акцентом.",
        open: "Відкрити профіль",
        diff: {
          E: "швидше виносить думку назовні й легше запускає контакт або дію.",
          I: "більше тримає процес усередині й потребує простору перед дією або розмовою.",
          S: "більше спирається на конкретику, перевірений досвід і видимі деталі.",
          N: "більше шукає зв'язки, можливості й майбутні сценарії.",
          T: "частіше відокремлює рішення від емоційного контексту й перевіряє логіку.",
          F: "більше зважає на людей, тон і наслідки рішення для стосунків.",
          J: "швидше закриває питання, структурує кроки й тримає план.",
          P: "довше тримає варіанти відкритими й легше підлаштовується по ходу.",
        },
      },
    },
    ru: {
      confidence: {
        title: "Уверенность результата",
        high: "Высокая",
        medium: "Средняя",
        close: "Близкий результат",
        highText: "Результат выглядит достаточно четким: большинство шкал заметно склоняются в свою сторону.",
        mediumText: "У результата есть рабочее направление, но одна из шкал близкая. Лучше читать тип как сильную гипотезу, а не как окончательный ярлык.",
        closeText: "Результат близкий: несколько шкал почти уравновешены. Похожие типы помогут точнее узнать свой стиль.",
        closest: "Самая близкая шкала: {axis} ({left} {leftScore} / {right} {rightScore}).",
      },
      why: {
        title: "Почему именно этот тип",
        intro: "Тип складывается из четырех небольших наклонов в ответах. Здесь коротко видно, какая сторона каждой шкалы проявилась чаще.",
        axis: {
          E: "Ответы чаще идут во внешний обмен: мысль легче проверять в разговоре, действии или живом контакте.",
          I: "Ответы чаще показывают внутреннюю обработку: сначала собрать мысль внутри, а уже потом говорить или действовать.",
          S: "Заметнее фокус на фактах, конкретике и том, что уже можно проверить на практике.",
          N: "Заметнее фокус на связях, смыслах и возможных сценариях развития.",
          T: "Решения чаще опираются на логику, критерии и желание понять, что именно работает.",
          F: "Решения чаще учитывают человеческое влияние, тон взаимодействия и ценность контакта.",
          J: "Ответы чаще тянутся к структуре: ясному плану, границам и завершению.",
          P: "Ответы чаще оставляют пространство для маневра, адаптации и живого процесса.",
        },
      },
      similar: {
        title: "Похожие типы",
        intro: "Эти типы близки по трем шкалам, но отличаются одним важным акцентом.",
        open: "Открыть профиль",
        diff: {
          E: "быстрее выносит мысль наружу и легче запускает контакт или действие.",
          I: "больше держит процесс внутри и нуждается в пространстве перед действием или разговором.",
          S: "больше опирается на конкретику, проверенный опыт и видимые детали.",
          N: "больше ищет связи, возможности и будущие сценарии.",
          T: "чаще отделяет решение от эмоционального контекста и проверяет логику.",
          F: "больше учитывает людей, тон и последствия решения для отношений.",
          J: "быстрее закрывает вопросы, структурирует шаги и держит план.",
          P: "дольше держит варианты открытыми и легче подстраивается по ходу.",
        },
      },
    },
    en: {
      confidence: {
        title: "Result confidence",
        high: "High",
        medium: "Medium",
        close: "Close result",
        highText: "The result looks fairly clear: most scales lean noticeably toward their selected side.",
        mediumText: "The result has a useful direction, but one scale is close. Treat the type as a strong hypothesis, not a final label.",
        closeText: "The result is close: several scales are nearly balanced. Similar types may help you recognize your pattern more precisely.",
        closest: "Closest scale: {axis} ({left} {leftScore} / {right} {rightScore}).",
      },
      why: {
        title: "Why this type",
        intro: "The type comes from four small leanings in your answers. This shows which side of each scale appeared more often.",
        axis: {
          E: "Your answers lean toward outer exchange: testing thoughts through conversation, action, or live contact.",
          I: "Your answers lean toward internal processing: forming the thought first, then speaking or acting.",
          S: "There is a stronger focus on facts, concrete details, and what can already be tested in practice.",
          N: "There is a stronger focus on patterns, meaning, and possible future scenarios.",
          T: "Decisions lean more on logic, criteria, and checking what actually works.",
          F: "Decisions lean more on human impact, interaction tone, and the value of connection.",
          J: "Your answers lean toward structure: a clear plan, boundaries, and completion.",
          P: "Your answers leave more room for adjustment, adaptation, and a live process.",
        },
      },
      similar: {
        title: "Similar types",
        intro: "These types share three scales with your result, but differ by one important accent.",
        open: "Open profile",
        diff: {
          E: "moves thoughts outward sooner and starts contact or action more easily.",
          I: "keeps more of the process internal and needs space before acting or speaking.",
          S: "leans more on concrete details, proven experience, and visible facts.",
          N: "looks more for patterns, possibilities, and future scenarios.",
          T: "separates decisions from emotional context more often and checks the logic.",
          F: "pays more attention to people, tone, and the relational impact of a decision.",
          J: "closes questions sooner, structures steps, and holds the plan more firmly.",
          P: "keeps options open longer and adjusts more easily as things unfold.",
        },
      },
    },
  };

  window.RESULT_INSIGHTS = Object.freeze({ text, similarAxisOrder });
})();
