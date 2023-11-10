# سكرول مونوريبو

[![rollup](https://github.com/scroll-tech/scroll/actions/workflows/rollup.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/rollup.yml)
[![contracts](https://github.com/scroll-tech/scroll/actions/workflows/contracts.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/contracts.yml)
[![bridge-history](https://github.com/scroll-tech/scroll/actions/workflows/bridge_history_api.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/bridge_history_api.yml)
[![coordinator](https://github.com/scroll-tech/scroll/actions/workflows/coordinator.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/coordinator.yml)
[![prover](https://github.com/scroll-tech/scroll/actions/workflows/prover.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/prover.yml)
[![integration](https://github.com/scroll-tech/scroll/actions/workflows/integration.yml/badge.svg)](https://github.com/scroll-tech/scroll/actions/workflows/integration.yml)
[![codecov](https://codecov.io/gh/scroll-tech/scroll/branch/develop/graph/badge.svg?token=VJVHNQWGGW)](https://codecov.io/gh/scroll-tech/scroll)

<a href="https://scroll.io">سكرول</a> هي تجميعه "ZK " للطبقه الثانيه  مخصصه لتعزيز قابلية التوسع في عمله ال  "الإيثريوم" من خلال ما يعادل دائره  "bytecode" [zkEVM]" (https://github.com/scroll-tech/zkevm-circuits) . يشمل هذا ال "monorepo" مكونات البنية التحتية الأساسية لبروتوكول "Scroll". يحتوي على عقود L1 و L2، وعقدة تجميعيه، وعميل مثبت ، ومنسق مثبت. 

## بنية الملف

<pre>
├── <a href="./bridge-history-api/">bridge-history-api</a>: خدمة تاريخ البريدج التي تجمع الإيداع وتسحب الأحداث من كل من سلسلة L1 و L2 وتولد أدلة السحب
├── <a href="./common/">common</a>:المكتبات والأنواع المشتركة
├── <a href="./coordinator/">coordinator</a>: خدمة منسق البروفر التي ترسل مهام إثبات إلى البروفرز
├── <a href="./database">database</a>: عملاء قاعدة البيانات وتعريف المخطط
├── <a href="./src">l2geth</a>:  "Scroll" نقطة تنفيذ
├── <a href="./prover">prover</a>: عميل البروفر الذي يدير توليد إثبات لدائرة "zkEVM" ودائرة التجميع
├── <a href="./rollup">rollup</a>: "Rollup"-الخدمات ذات الصلة ب
├── <a href="./rpc-gateway">rpc-gateway</a>: "RPC" إعادة الشراء الخارجية للبوابة
└── <a href="./tests">tests</a>: اختبارات الدمج
</pre>

## المساهمة 

نرحب بمساهمات المجتمع في هذا المستودع. قبل إرسال أي مشكلات أو علاقات عامة، يرجى قراءة [كود الإجراء ](CODE_OF_CONDUCT.md) and the [المبادئ التوجيهية للمساهمات](CONTRIBUTING.md).

## الشروط المبدائيه
+ "Go" 1.19
+ "Rust" (لل الإصدار, اذهب الي [rust-toolchain](./common/libzkp/impl/rust-toolchain))
+ "Hardhat" / "Foundry"
+ "Docker"

لإجراء الاختبارات، من الضروري أولاً سحب أو بناء صور "Docker" المطلوبة. نفذ الأوامر التالية في ملف الروت المستودع للقيام بذلك: 

```bash
docker pull postgres
make dev_docker
```

## تجربة "Rollup" و "Coordinator"

### ل أجهزه آبل غير سيليكون  (M1/M2) ماك

قم بإجراء الاختبارات باستخدام الأوامر التالية:

```bash
go test -v -race -covermode=atomic scroll-tech/rollup/...
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
go test -v -race -covermode=atomic scroll-tech/database/...
go test -v -race -covermode=atomic scroll-tech/common/...
```

### ل أجهزه ابل سيليكون  (M1/M2) ماك

لإجراء الإختبارات على  اجهزه آبل ماك سيليكون، قم ببناء وتنفيذ صورة "Docker" على النحو المبين التالي: 

#### بناء صورة "Docker" للاختبار 

استخدم الأمر التالي لبناء صورة "Docker"

```bash
make build_test_docker
```

هذا الأمر يبني صورة "Docker" تحت اسم `scroll_test_image` استخدام "Dockerfile" الموجود على `./build/dockerfiles/local_test.Dockerfile`.

#### قم بتشغيل صورة "Docker"

بعد بناء الصورة، قم بتشغيل "Docker Container"  منها: 

```bash
make run_test_docker
```

ذا الأمر يفعل Docker container اسمها  `scroll_test_container` من صوره. `scroll_test_image` . ستخدم الحاوية الشبكة المضيفة ولديها إمكانية الوصول إلى مقبس "Docker" والملف الحالي

بمجرد تشغيل ال "Docker container" ، قم بتنفيذ الاختبارات باستخدام الأوامر التالية:

```bash
go test -v -race -covermode=atomic scroll-tech/rollup/...
go test -tags="mock_verifier" -v -race -covermode=atomic scroll-tech/coordinator/...
go test -v -race -covermode=atomic scroll-tech/database/...
go test -v -race -covermode=atomic scroll-tech/common/...
```

##   اختبار العقود

يمكنك العثور على اختبارات الوحدة في [`contracts/src/test/`](/contracts/src/test/), واختبارات الدمج في [`contracts/integration-test/`](/contracts/integration-test/).

أذهب الي [`contracts`](/contracts) لمزيد من التفاصيل حول العقود.

## الترخيص

تم ترخيص سكرول مونوريبو تحت رخصه [MIT](./LICENSE) .
