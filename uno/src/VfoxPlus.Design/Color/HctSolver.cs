// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

namespace VfoxPlus.Design.Color;

/// <summary>
/// The HCT solver of the Material colour utilities (Copyright 2021 Google LLC, Apache License 2.0): finds the
/// sRGB colour closest to a hue, a chroma and a tone (L*), by first trying to solve for the CAM16 lightness
/// and then bisecting along the edges of the sRGB gamut.
/// </summary>
internal static class HctSolver
{
    private static readonly double[][] ScaledDiscountFromLinRgb =
    [
        [0.001200833568784504, 0.002389694492170889, 0.0002795742885861124],
        [0.0005891086651375999, 0.0029785502573438758, 0.0003270666104008398],
        [0.00010146692491640572, 0.0005364214359186694, 0.0032979401770712076],
    ];

    private static readonly double[][] LinRgbFromScaledDiscount =
    [
        [1373.2198709594231, -1100.4251190754821, -7.278681089101213],
        [-271.815969077903, 559.6580465940733, -32.46047482791194],
        [1.9622899599665666, -57.173814538844006, 308.7233197812385],
    ];

    private static readonly double[] YFromLinRgb = [0.2126, 0.7152, 0.0722];

    // The linear RGB values (0 to 100) at which the colour channels change from one 8-bit value to the
    // next: the planes along which the HCT solver bisects. They are the values of the CRITICAL_PLANES table
    // of the Material colour utilities.
    private static readonly double[] CriticalPlanes =
    [
        0.015176349177441876, 0.045529047532325624, 0.07588174588720938,
        0.10623444424209313, 0.13658714259697685, 0.16693984095186062,
        0.19729253930674434, 0.2276452376616281, 0.2579979360165119,
        0.28835063437139563, 0.3188300904430532, 0.350925934958123,
        0.3848314933096426, 0.42057480301049466, 0.458183274052838,
        0.4976837250274023, 0.5391024159806381, 0.5824650784040898,
        0.6277969426914107, 0.6751227633498623, 0.7244668422128921,
        0.775853049866786, 0.829304845476233, 0.8848452951698498,
        0.942497089126609, 1.0022825574869039, 1.0642236851973577,
        1.1283421258858297, 1.1946592148522128, 1.2631959812511864,
        1.3339731595349034, 1.407011200216447, 1.4823302800086415,
        1.5599503113873272, 1.6398909516233677, 1.7221716113234105,
        1.8068114625156377, 1.8938294463134073, 1.9832442801866852,
        2.075074464868551, 2.1693382909216234, 2.2660538449872063,
        2.36523901573795, 2.4669114995532007, 2.5710888059345764,
        2.6777882626779785, 2.7870270208169257, 2.898822059350997,
        3.0131901897720907, 3.1301480604002863, 3.2497121605402226,
        3.3718988244681087, 3.4967242352587946, 3.624204428461639,
        3.754355295633311, 3.887192587735158, 4.022731918402185,
        4.160988767090289, 4.301978482107941, 4.445716283538092,
        4.592217266055746, 4.741496401646282, 4.893568542229298,
        5.048448422192488, 5.20615066083972, 5.3666897647573375,
        5.5300801301023865, 5.696336044816294, 5.865471690767354,
        6.037501145825082, 6.212438385869475, 6.390297286737924,
        6.571091626112461, 6.7548350853498045, 6.941541251256611,
        7.131223617812143, 7.323895587840543, 7.5195704746346665,
        7.7182615035334345, 7.919981813454504, 8.124744458384042,
        8.332562408825165, 8.543448553206703, 8.757415699253682,
        8.974476575321063, 9.194643831691977, 9.417930041841839,
        9.644347703669503, 9.873909240696694, 10.106627003236781,
        10.342513269534024, 10.58158024687427, 10.8238400726681,
        11.069304815507364, 11.317986476196008, 11.569896988756009,
        11.825048221409341, 12.083451977536606, 12.345119996613247,
        12.610063955123938, 12.878295467455942, 13.149826086772048,
        13.42466730586372, 13.702830557985108, 13.984327217668513,
        14.269168601521828, 14.55736596900856, 14.848930523210871,
        15.143873411576273, 15.44220572664832, 15.743938506781891,
        16.04908273684337, 16.35764934889634, 16.66964922287304,
        16.985093187232053, 17.30399201960269, 17.62635644741625,
        17.95219714852476, 18.281524751807332, 18.614349837764564,
        18.95068293910138, 19.290534541298456, 19.633915083172692,
        19.98083495742689, 20.331304511189067, 20.685334046541502,
        21.042933821039977, 21.404114048223256, 21.76888489811322,
        22.137256497705877, 22.50923893145328, 22.884842241736916,
        23.264076429332462, 23.6469514538663, 24.033477234264016,
        24.42366364919083, 24.817520537484558, 25.21505769858089,
        25.61628489293138, 26.021211842414342, 26.429848230738664,
        26.842203703840827, 27.258287870275353, 27.678110301598522,
        28.10168053274597, 28.529008062403893, 28.96010235337422,
        29.39497283293396, 29.83362889318845, 30.276079891419332,
        30.722335150426627, 31.172403958865512, 31.62629557157785,
        32.08401920991837, 32.54558406207592, 33.010999283389665,
        33.4802739966603, 33.953417292456834, 34.430438229418264,
        34.911345834551085, 35.39614910352207, 35.88485700094671,
        36.37747846067349, 36.87402238606382, 37.37449765026789,
        37.87891309649659, 38.38727753828926, 38.89959975977785,
        39.41588851594697, 39.93615253289054, 40.460400508064545,
        40.98864111053629, 41.520882981230194, 42.05713473317016,
        42.597404951718396, 43.141702194811224, 43.6900349931913,
        44.24241185063697, 44.798841244188324, 45.35933162437017,
        45.92389141541209, 46.49252901546552, 47.065252796817916,
        47.64207110610409, 48.22299226451468, 48.808024568002054,
        49.3971762874833, 49.9904556690408, 50.587870934119984,
        51.189430279724725, 51.79514187861014, 52.40501387947288,
        53.0190544071392, 53.637271562750364, 54.259673423945976,
        54.88626804504493, 55.517063457223934, 56.15206766869424,
        56.79128866487574, 57.43473440856916, 58.08241284012621,
        58.734331877617365, 59.39049941699807, 60.05092333227251,
        60.715611475655585, 61.38457167773311, 62.057811747619894,
        62.7353394731159, 63.417162620860914, 64.10328893648692,
        64.79372614476921, 65.48848194977529, 66.18756403501224,
        66.89098006357258, 67.59873767827808, 68.31084450182222,
        69.02730813691093, 69.74813616640164, 70.47333615344107,
        71.20291564160104, 71.93688215501312, 72.67524319850172,
        73.41800625771542, 74.16517879925733, 74.9167682708136,
        75.67278210128072, 76.43322770089146, 77.1981124613393,
        77.96744375590167, 78.74122893956174, 79.51947534912904,
        80.30219030335869, 81.08938110306934, 81.88105503125999,
        82.67721935322541, 83.4778813166706, 84.28304815182372,
        85.09272707154808, 85.90692527145302, 86.72564993000343,
        87.54890820862819, 88.3767072518277, 89.2090541872801,
        90.04595612594655, 90.88742016217518, 91.73345337380438,
        92.58406282226491, 93.43925555268066, 94.29903859396902,
        95.16341895893969, 96.03240364439274, 96.9059996312159,
        97.78421388448044, 98.6670533535366, 99.55452497210776,
    ];

    /// <summary>
    /// Returns the sRGB colour with the given hue (degrees), chroma and tone (L*), or the closest one inside
    /// the gamut.
    /// </summary>
    /// <param name="hueDegrees">The hue.</param>
    /// <param name="chroma">The chroma.</param>
    /// <param name="lstar">The tone.</param>
    /// <returns>The colour as ARGB.</returns>
    internal static uint SolveToArgb(double hueDegrees, double chroma, double lstar)
    {
        if (chroma < 0.0001 || lstar < 0.0001 || lstar > 99.9999)
        {
            return ColorMath.ArgbFromLstar(lstar);
        }

        hueDegrees = ColorMath.SanitizeDegrees(hueDegrees);
        double hueRadians = hueDegrees / 180 * Math.PI;
        double y = ColorMath.YFromLstar(lstar);
        uint exact = FindResultByJ(hueRadians, chroma, y);
        return exact != 0 ? exact : ColorMath.ArgbFromLinRgb(BisectToLimit(y, hueRadians));
    }

    private static double SanitizeRadians(double angle) => (angle + (Math.PI * 8)) % (Math.PI * 2);

    private static double ChromaticAdaptation(double component)
    {
        double af = Math.Pow(Math.Abs(component), 0.42);
        return ColorMath.Signum(component) * 400 * af / (af + 27.13);
    }

    // The hue, in radians, of a linear RGB colour.
    private static double HueOf(double[] linrgb)
    {
        double[] scaled = ColorMath.MatrixMultiply(linrgb, ScaledDiscountFromLinRgb);
        double rA = ChromaticAdaptation(scaled[0]);
        double gA = ChromaticAdaptation(scaled[1]);
        double bA = ChromaticAdaptation(scaled[2]);
        double a = ((11 * rA) + (-12 * gA) + bA) / 11;
        double b = (rA + gA - (2 * bA)) / 9;
        return Math.Atan2(b, a);
    }

    private static bool AreInCyclicOrder(double a, double b, double c) =>
        SanitizeRadians(b - a) < SanitizeRadians(c - a);

    // How far mid lies between source and target.
    private static double Intercept(double source, double mid, double target) => (mid - source) / (target - source);

    private static double[] LerpPoint(double[] source, double t, double[] target) =>
    [
        source[0] + ((target[0] - source[0]) * t),
        source[1] + ((target[1] - source[1]) * t),
        source[2] + ((target[2] - source[2]) * t),
    ];

    // The point between source and target whose coordinate on an axis is the given one.
    private static double[] SetCoordinate(double[] source, double coordinate, double[] target, int axis) =>
        LerpPoint(source, Intercept(source[axis], coordinate, target[axis]), target);

    private static bool IsBounded(double x) => x is >= 0 and <= 100;

    // The nth vertex of the intersection of the plane of the given luminance and the RGB cube, or
    // (-1, -1, -1) if there is none.
    private static double[] NthVertex(double y, int n)
    {
        double kR = YFromLinRgb[0];
        double kG = YFromLinRgb[1];
        double kB = YFromLinRgb[2];
        double coordA = n % 4 <= 1 ? 0 : 100.0;
        double coordB = n % 2 == 0 ? 0 : 100.0;
        if (n < 4)
        {
            double g = coordA;
            double b = coordB;
            double r = (y - (g * kG) - (b * kB)) / kR;
            if (IsBounded(r))
            {
                return [r, g, b];
            }
        }
        else if (n < 8)
        {
            double b = coordA;
            double r = coordB;
            double g = (y - (r * kR) - (b * kB)) / kG;
            if (IsBounded(g))
            {
                return [r, g, b];
            }
        }
        else
        {
            double r = coordA;
            double g = coordB;
            double b = (y - (r * kR) - (g * kG)) / kB;
            if (IsBounded(b))
            {
                return [r, g, b];
            }
        }

        return [-1, -1, -1];
    }

    // The segment of the polygon of equal luminance that contains the target hue.
    private static (double[] Left, double[] Right) BisectToSegment(double y, double targetHue)
    {
        double[] left = [-1, -1, -1];
        double[] right = left;
        double leftHue = 0;
        double rightHue = 0;
        bool initialized = false;
        bool uncut = true;
        for (int n = 0; n < 12; n++)
        {
            double[] mid = NthVertex(y, n);
            if (mid[0] < 0)
            {
                continue;
            }

            double midHue = HueOf(mid);
            if (!initialized)
            {
                left = mid;
                right = mid;
                leftHue = midHue;
                rightHue = midHue;
                initialized = true;
                continue;
            }

            if (uncut || AreInCyclicOrder(leftHue, midHue, rightHue))
            {
                uncut = false;
                if (AreInCyclicOrder(leftHue, targetHue, midHue))
                {
                    right = mid;
                    rightHue = midHue;
                }
                else
                {
                    left = mid;
                    leftHue = midHue;
                }
            }
        }

        return (left, right);
    }

    private static double[] Midpoint(double[] a, double[] b) =>
        [(a[0] + b[0]) / 2, (a[1] + b[1]) / 2, (a[2] + b[2]) / 2];

    private static int CriticalPlaneBelow(double x) => (int)Math.Floor(x - 0.5);

    private static int CriticalPlaneAbove(double x) => (int)Math.Ceiling(x - 0.5);

    // A colour with the given luminance and hue that is as chromatic as possible within the gamut, to the
    // precision of the critical planes.
    private static double[] BisectToLimit(double y, double targetHue)
    {
        (double[] left, double[] right) = BisectToSegment(y, targetHue);
        double leftHue = HueOf(left);
        for (int axis = 0; axis < 3; axis++)
        {
            if (left[axis] == right[axis])
            {
                continue;
            }

            int lPlane;
            int rPlane;
            if (left[axis] < right[axis])
            {
                lPlane = CriticalPlaneBelow(ColorMath.TrueDelinearized(left[axis]));
                rPlane = CriticalPlaneAbove(ColorMath.TrueDelinearized(right[axis]));
            }
            else
            {
                lPlane = CriticalPlaneAbove(ColorMath.TrueDelinearized(left[axis]));
                rPlane = CriticalPlaneBelow(ColorMath.TrueDelinearized(right[axis]));
            }

            for (int i = 0; i < 8; i++)
            {
                if (Math.Abs(rPlane - lPlane) <= 1)
                {
                    break;
                }

                int mPlane = (int)Math.Floor((lPlane + rPlane) / 2.0);
                double[] mid = SetCoordinate(left, CriticalPlanes[mPlane], right, axis);
                double midHue = HueOf(mid);
                if (AreInCyclicOrder(leftHue, targetHue, midHue))
                {
                    right = mid;
                    rPlane = mPlane;
                }
                else
                {
                    left = mid;
                    leftHue = midHue;
                    lPlane = mPlane;
                }
            }
        }

        return Midpoint(left, right);
    }

    private static double InverseChromaticAdaptation(double adapted)
    {
        double adaptedAbs = Math.Abs(adapted);
        double basis = Math.Max(0, 27.13 * adaptedAbs / (400 - adaptedAbs));
        return ColorMath.Signum(adapted) * Math.Pow(basis, 1.0 / 0.42);
    }

    // Solves for the colour of the given hue, chroma and luminance by iterating on the CAM16 lightness J. It
    // returns 0 if there is none inside the gamut.
    private static uint FindResultByJ(double hueRadians, double chroma, double y)
    {
        double j = Math.Sqrt(y) * 11;
        Cam16.ViewingConditions vc = Cam16.Default;
        double tInnerCoeff = 1 / Math.Pow(1.64 - Math.Pow(0.29, vc.N), 0.73);
        double eHue = 0.25 * (Math.Cos(hueRadians + 2) + 3.8);
        double p1 = eHue * (50000.0 / 13.0) * vc.Nc * vc.Ncb;
        double hSin = Math.Sin(hueRadians);
        double hCos = Math.Cos(hueRadians);
        for (int round = 0; round < 5; round++)
        {
            double jNormalized = j / 100;
            double alpha = chroma != 0 && j != 0 ? chroma / Math.Sqrt(jNormalized) : 0.0;
            double t = Math.Pow(alpha * tInnerCoeff, 1.0 / 0.9);
            double ac = vc.Aw * Math.Pow(jNormalized, 1 / vc.C / vc.Z);
            double p2 = ac / vc.Nbb;
            double gamma = 23 * (p2 + 0.305) * t / ((23 * p1) + (11 * t * hCos) + (108 * t * hSin));
            double a = gamma * hCos;
            double b = gamma * hSin;
            double rA = ((460 * p2) + (451 * a) + (288 * b)) / 1403;
            double gA = ((460 * p2) - (891 * a) - (261 * b)) / 1403;
            double bA = ((460 * p2) - (220 * a) - (6300 * b)) / 1403;
            double[] linrgb = ColorMath.MatrixMultiply(
                [InverseChromaticAdaptation(rA), InverseChromaticAdaptation(gA), InverseChromaticAdaptation(bA)],
                LinRgbFromScaledDiscount);
            if (linrgb[0] < 0 || linrgb[1] < 0 || linrgb[2] < 0)
            {
                return 0;
            }

            double fnj = (YFromLinRgb[0] * linrgb[0]) + (YFromLinRgb[1] * linrgb[1]) + (YFromLinRgb[2] * linrgb[2]);
            if (fnj <= 0)
            {
                return 0;
            }

            if (round == 4 || Math.Abs(fnj - y) < 0.002)
            {
                return linrgb[0] > 100.01 || linrgb[1] > 100.01 || linrgb[2] > 100.01 ? 0 : ColorMath.ArgbFromLinRgb(linrgb);
            }

            // Iterate with Newton's method on the lightness.
            j -= (fnj - y) * j / (2 * fnj);
        }

        return 0;
    }
}
